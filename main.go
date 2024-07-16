package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	urlverifier "github.com/davidmytton/url-verifier"
	"github.com/gofiber/fiber/v2"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/redis/go-redis/v9"
)

var (
	PORT      = os.Getenv("PORT")
	REDIS_URI = os.Getenv("REDIS_URI")
)

func init() {
	if redisURI := os.Getenv("REDIS_URI"); redisURI != "" {
		REDIS_URI = redisURI
	}

	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			panic(err)
		}

		PORT = fmt.Sprintf(":%d", p)
	}
}

func validateURL(verifier urlverifier.Verifier, url string) bool {
	ret, err := verifier.Verify(url)
	if err != nil {
		return false
	}
	return ret.IsURL
}

func main() {
	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})
	verifier := *urlverifier.NewVerifier()

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

		if !validateURL(verifier, body.URL) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid URL",
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
			"id": id,
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
