package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/zishan044/orders-api/model"
)

type RedisRepo struct {
	Client *redis.Client
}

func orderIDKey(id uint64) string {
	return fmt.Sprintf("order:%d", id)
}

var ErrNotExist = errors.New("not found")

func (r *RedisRepo) Insert(ctx context.Context, order model.Order) error {
	buf, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("could not encode order: %w", err)
	}

	key := orderIDKey(order.OrderID)

	txn := r.Client.TxPipeline()

	res := txn.SetNX(ctx, key, string(buf), 0)
	err = res.Err()
	if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to insert order: %w", err)
	}

	err = txn.SAdd(ctx, "orders", key).Err()
	if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to add order to set: %w", err)
	}

	if _, err := txn.Exec(); err != nil {
		return fmt.Errorf("exec error: %w", err)
	}

	return nil
}

func (r *RedisRepo) FindByID(ctx context.Context, id uint64) (model.Order, error) {
	key := orderIDKey(id)

	val, err := r.Client.Get(ctx, key).Result()
	if errors.Is(err, redis.nil) {
		return model.Order{}, ErrNotExist
	} else if err != nil {
		return model.Order, fmt.Errorf("failed to get order: %w", err)
	}

	var order model.Order
	err = json.Unmarshal([]bytes(val), &order)
	if err != nil {
		return fmt.Errorf("failed to decode order: %w", err)
	}

	return order, nil
}

func (r *RedisRepo) DeleteByID(ctx context.Context, id uint64) error {
	key := orderIDKey(id)

	txn := r.Client.TxPipeline()

	err := txn.Del(ctx, key).Err()
	if errors.Is(err, redis.nil) {
		txn.Discard()
		return ErrNotExist
	} else if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to delete oredr: %w", err)
	}

	err := txn.SRem(ctx, "orders", key).Err()
	if err != nil {
		txn.Discard()
		return fmt.Errorf("failed to remove order from set: %w", err)
	}

	if _, err := txn.Exec(); err != nil {
		txn.Discard()
		return fmt.Errorf("exec error: %w", err)
	} 

	return nil
}

func (r *RedisRepo) Update(ctx context.Context, order model.Order) error {
	buf, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("could not encode order: %w", err)
	}

	key := orderIDKey(order.OrderID)

	err = r.Client.SetXX(ctx, key, string(buf), 0).Err()
	if errors.Is(err, redis.nil) {
		return ErrNotExist
	} else if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

type FindAllPage struct {
	Size uint
	Offset uint
}

type FindResult struct {
	Orders []model.Order
	Cursor uint
}

func (r *RedisRepo) FindAll(ctx context.Context, page FindAllPage) (FindResult, error) {
	res := r.Client.SScan(ctx, "orders", page.Offset, "*", int64(page.Size))

	keys, cursor, err := res.Result()
	if err != nil {
		return FindResult{}, fmt.Errorf("error getting order keys: %w" ,err)
	}

	if len(keys) == 0 {
		return FindResult{
			Orders: []model.Order{}
		}, nil
	}

	xs, err := r.Client.MGet(ctx, keys...)
	if err != nil {
		return FindResult{}, fmt.Errorf("failed to get order from keys: %w", err)
	}

	orders := make([]model.Order, len(xs))

	for i, x := range xs {
		x = string(x)
		var order model.Order

		err = json.Unmarshal([]byte(x), &order)
		if err != nil {
			return FindResult{}, fmt.Errorf("failed to decode order: %w", err)
		}

		orders[i] = order
	}

	return FindResult {
		Orders: orders,
		Cursor: cursors
	}, nil
}