package blog

import (
	"comics-galore-web/internal/database"
	"context"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"time"
)

func SmartViewTracker(svc Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		// 1. Skip if already rate limited by the previous middleware
		if c.Locals("is_rate_limited") == true {
			return c.Next()
		}

		postIDStr := c.Params("id")
		postID, err := uuid.Parse(postIDStr)
		if err != nil {
			return c.Next()
		}

		// 2. Refresh Spam Check: Check for a unique cookie for this post
		cookieName := "v_limit_" + postID.String()[:8]
		if c.Cookies(cookieName) != "" {
			return c.Next() // Already viewed in the last hour, skip DB update
		}

		// 3. Set the cookie to prevent spam for the next hour
		c.Cookie(&fiber.Cookie{
			Name:     cookieName,
			Value:    "1",
			Expires:  time.Now().Add(1 * time.Hour),
			HTTPOnly: true, // Security best practice
			SameSite: "Lax",
		})

		// 4. Proceed with the request
		if err := c.Next(); err != nil {
			return err
		}

		// 5. Async DB Update (same as before)
		if c.Response().StatusCode() == fiber.StatusOK {
			isAuth := c.Locals("user") != nil
			go func(id uuid.UUID, auth bool) {
				authInc, anonInc := int64(0), int64(0)
				if auth {
					authInc = 1
				} else {
					anonInc = 1
				}
				_ = svc.IncrementView(context.Background(), database.IncrementPostStatsParams{
					PostID:    id,
					AuthViews: authInc,
					AnonViews: anonInc,
				})
			}(postID, isAuth)
		}
		return nil
	}
}

/*
func RedisViewTracker(rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		// ... (Keep the Rate Limiter and Cookie checks from the previous step) ...

		if c.Response().StatusCode() == fiber.StatusOK {
			postID := c.Params("id")
			isAuth := c.Locals("user") != nil

			// We use a Redis Hash for efficiency
			// Key: "v_stats_auth" or "v_stats_anon" | Field: {post_id} | Value: increment
			field := "v_stats_anon"
			if isAuth {
				field = "v_stats_auth"
			}

			go func() {
				// HINCRBY is atomic and extremely fast
				rdb.HIncrBy(context.Background(), field, postID, 1)
			}()
		}
		return nil
	}
}
*/
