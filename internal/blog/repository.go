package blog

/*func StartViewSyncWorker(rdb *redis.Client, queries *db.Queries, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			ctx := context.Background()

			// Sync both Auth and Anon hashes
			syncHash(ctx, rdb, queries, "v_stats_auth", true)
			syncHash(ctx, rdb, queries, "v_stats_anon", false)
		}
	}()
}

func syncHash(ctx context.Context, rdb *redis.Client, queries *db.Queries, hashKey string, isAuth bool) {
	// 1. Get all pending views from Redis
	data, err := rdb.HGetAll(ctx, hashKey).Result()
	if err != nil || len(data) == 0 {
		return
	}

	// 2. Loop through and update PostgreSQL
	for postIDStr, countStr := range data {
		postID, _ := uuid.Parse(postIDStr)
		count, _ := strconv.ParseInt(countStr, 10, 64)

		params := db.IncrementPostViewParams{PostID: postID}
		if isAuth {
			params.AuthViews = count
		} else {
			params.AnonViews = count
		}

		_ = queries.IncrementPostView(ctx, params)
	}

	// 3. Clear the hash so we don't double-count next time
	// Note: In extreme traffic, use a "Rename and Process" pattern to avoid race conditions
	rdb.Del(ctx, hashKey)
}*/
