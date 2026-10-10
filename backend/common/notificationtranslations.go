package common

import (
	"fmt"
	"strings"
)

// spanishNotificationTemplates mirrors frontend src/i18n/locales/es.json's notifications.messages
// block. It exists because deliverExternalNotifications (email/push) runs in the Go backend,
// which has no access to the frontend's i18n JSON files at runtime - so translations that need to
// reach an external channel are duplicated here by hand, same convention as common.RegionForCountry
// mirroring frontend regionForCountry. Keep this in sync with es.json's notifications.messages.
//
// Placeholders use the same {{name}} mustache syntax as the frontend's i18next config, and are
// substituted by renderNotificationMessage - simple string replacement, no pluralization rules,
// since every key used here has identical singular/plural text in both languages.
var spanishNotificationTemplates = map[string]string{
	"notifications.messages.trainerHired":           "Fichado {{name}} como entrenador",
	"notifications.messages.academyPromotion":       "{{name}} promocionado de la cantera al primer equipo",
	"notifications.messages.transferSold":           "Vendido {{player}} a {{team}}",
	"notifications.messages.transferBought":         "Comprado {{player}} de {{team}}",
	"notifications.messages.cupChampion":            "¡{{team}} ha ganado la copa!",
	"notifications.messages.cupTieResult":           "Copa: {{home}} {{homeGoals}} - {{awayGoals}} {{away}}",
	"notifications.messages.cupTieResultPenalties":  "Copa: {{home}} {{homeGoals}} - {{awayGoals}} {{away}} (penaltis)",
	"notifications.messages.freeAgentSigned":        "Fichado el agente libre {{name}} por {{years}} temporada(s)",
	"notifications.messages.freeAgentPremium":       "{{name}} ha sido fichado por otro club - prima de traspaso recibida",
	"notifications.messages.contractRenewed":        "{{name}} aceptó una renovación de contrato de {{years}} temporada(s)",
	"notifications.messages.loanedIn":               "Cedido {{player}} desde {{team}}",
	"notifications.messages.loanedOut":              "Cedido {{player}} a {{team}}",
	"notifications.messages.loanReturnedHome":       "{{name}} ha vuelto de su cesión",
	"notifications.messages.loanReturnedAway":       "{{name}} ha vuelto a {{team}}",
	"notifications.messages.promoted":               "¡{{team}} ha ascendido!",
	"notifications.messages.relegated":              "¡{{team}} ha descendido!",
	"notifications.messages.matchResult":            "{{home}} {{homeGoals}} - {{awayGoals}} {{away}}",
	"notifications.messages.friendlyResult":         "Amistoso: {{home}} {{homeGoals}} - {{awayGoals}} {{away}}",
	"notifications.messages.playerEvolutions":       "{{count}} de tus jugadores han evolucionado esta temporada - revisa sus últimas evoluciones",
	"notifications.messages.goldenBoot":             "¡{{name}} ha ganado la Bota de Oro con {{goals}} goles!",
	"notifications.messages.playerOfSeason":         "¡{{name}} ha sido nombrado Jugador de la Temporada!",
	"notifications.messages.fanbaseGrew":            "Tu afición ha crecido a {{fanbase}}",
	"notifications.messages.fanbaseShrank":          "Tu afición ha bajado a {{fanbase}}",
	"notifications.messages.loanBuyOptionExercised": "{{team}} ha ejecutado la opción de compra de {{player}}",
	"notifications.messages.loanBuyOptionUsed":      "Fichado {{player}} de forma permanente desde {{team}}",
	"notifications.messages.calledUp":               "{{player}} ha sido convocado por {{team}} para el amistoso internacional de esta semana",
}

// renderNotificationMessage renders msg in the given language for external delivery (email/push),
// falling back to msg.Text (English) when the language isn't Spanish, the message has no key, or
// the key is unrecognized - see deliverExternalNotifications.
func renderNotificationMessage(msg NotificationMessage, language string) string {
	if language != "es" || msg.Key == "" {
		return msg.Text
	}
	template, ok := spanishNotificationTemplates[msg.Key]
	if !ok {
		return msg.Text
	}
	rendered := template
	for param, value := range msg.Params {
		rendered = strings.ReplaceAll(rendered, "{{"+param+"}}", fmt.Sprintf("%v", value))
	}
	return rendered
}
