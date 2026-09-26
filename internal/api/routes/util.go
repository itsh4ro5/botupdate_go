package routes

import "github.com/gofiber/fiber/v2"

func getStringLocalSafe(c *fiber.Ctx, key string) (string, bool) {
	val, ok := c.Locals(key).(string)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}
