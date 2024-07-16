package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/redis/go-redis/v9"
)

var (
	PORT      = os.Getenv("PORT")
	REDIS_URI = os.Getenv("REDIS_URI")
	BASE_URL  = os.Getenv("BASE_URL")
)

func init() {
	if redisURI := os.Getenv("REDIS_URI"); redisURI != "" {
		REDIS_URI = redisURI
	}

	if baseUrl := os.Getenv("BASE_URL"); baseUrl != "" {
		BASE_URL = baseUrl
	}

	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			panic(err)
		}

		PORT = fmt.Sprintf(":%d", p)
	}
}

func main() {
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	redisClient := redis.NewClient(&redis.Options{
		Addr: REDIS_URI,
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		panic(err)
	}

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	app.Post("/new", func(c *fiber.Ctx) error {
		body := struct {
			URL string `json:"url"`
		}{}

		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		} else if body.URL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "URL is required",
			})
		}

		id, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz1234567890", 5)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		redisClient.Set(c.Context(), id, body.URL, 0)

		return c.JSON(fiber.Map{
			"id":   id,
			"link": fmt.Sprintf("%s/%s", BASE_URL, id),
		})
	})

	app.Get("/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		url, err := redisClient.Get(c.Context(), id).Result()
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "URL not found",
			})
		}

		return c.Redirect(url, fiber.StatusMovedPermanently)
	})

	app.Listen(PORT)
}
