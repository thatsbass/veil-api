package gateway

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thatsbass/veil/internal/translator"
	"github.com/thatsbass/veil/pkg/models"
)

// detectFormat infers the request format from the endpoint path.
func detectFormat(c *fiber.Ctx) models.RequestFormat {
	switch c.Path() {
	case "/v1/messages":
		return models.FormatAnthropic
	case "/v1/responses":
		return models.FormatResponses
	default:
		return models.FormatOpenAI
	}
}

// selectTranslator returns the right translator for the inbound format.
func selectTranslator(format models.RequestFormat, t *translators) translator.Translator {
	switch format {
	case models.FormatAnthropic:
		return t.anthropic
	case models.FormatResponses:
		return t.responses
	default:
		return t.openai
	}
}

type translators struct {
	anthropic translator.Translator
	openai    translator.Translator
	responses translator.Translator
}
