package cache

import (
	"context"
	"fmt"
	"time"
)

const contactTTL = 24 * time.Hour

func StoreUserContact(userID int64, email, phone string) error {
	key := fmt.Sprintf("user_contact:%d", userID)
	data := map[string]interface{}{
		"email": email,
		"phone": phone,
	}
	err := Rdb.HSet(context.Background(), key, data).Err()
	if err != nil {
		return err
	}
	return Rdb.Expire(context.Background(), key, contactTTL).Err()
}

func GetUserContact(userID int64) (email string, phone string, err error) {
	key := fmt.Sprintf("user_contact:%d", userID)
	data, err := Rdb.HGetAll(context.Background(), key).Result()
	if err != nil {
		return "", "", err
	}
	email = data["email"]
	phone = data["phone"]
	return email, phone, nil
}
