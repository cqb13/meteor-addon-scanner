package discord

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type WebhookPayload struct {
	Username  string   `json:"username"`
	AvatarUrl string   `json:"avatar_url"`
	Content   string   `json:"content"`
	Embeds    []*Embed `json:"embeds"`
}

func NewWebhookPayload() *WebhookPayload {
	return &WebhookPayload{}
}

func (p *WebhookPayload) WithUsername(username string) *WebhookPayload {
	p.Username = username
	return p
}

func (p *WebhookPayload) WithAvatarURl(url string) *WebhookPayload {
	p.AvatarUrl = url
	return p
}

func (p *WebhookPayload) WithContent(content string) *WebhookPayload {
	p.Content = content
	return p
}

func (p *WebhookPayload) AddEmbed(embed *Embed) *WebhookPayload {
	p.Embeds = append(p.Embeds, embed)
	return p
}

func (p *WebhookPayload) ClearEmbeds() *WebhookPayload {
	clear(p.Embeds)
	return p
}

type Embed struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Url         string       `json:"url"`
	Color       int64        `json:"color"`
	Timestamp   string       `json:"timestamp"`
	Footer      EmbedFooter  `json:"footer"`
	Image       EmbedImage   `json:"image"`
	Thumbnail   EmbedImage   `json:"thumbnail"`
	Author      EmbedAuthor  `json:"author"`
	Fields      []EmbedField `json:"fields"`
}

func NewEmbed() *Embed {
	return &Embed{}
}

func (e *Embed) WithTitle(title string) *Embed {
	e.Title = title
	return e
}

func (e *Embed) WithDescription(description string) *Embed {
	e.Description = description
	return e
}

func (e *Embed) WithUrl(url string) *Embed {
	e.Url = url
	return e
}

func (e *Embed) WithHexColor(color string) *Embed {
	e.Color = HexToDecimal(color)
	return e
}

func (e *Embed) WithColor(color int64) *Embed {
	e.Color = color
	return e
}

func (e *Embed) WithTimestamp(timestamp string) *Embed {
	e.Timestamp = timestamp
	return e
}

func (e *Embed) WithFooter(text, iconUrl string) *Embed {
	e.Footer = EmbedFooter{
		text,
		iconUrl,
	}
	return e
}

func (e *Embed) WithImage(url string) *Embed {
	e.Image = EmbedImage{
		url,
	}
	return e
}

func (e *Embed) WithThumbnail(url string) *Embed {
	e.Thumbnail = EmbedImage{
		url,
	}
	return e
}

func (e *Embed) WithAuthor(name, url, iconUrl string) *Embed {
	e.Author = EmbedAuthor{
		name,
		url,
		iconUrl,
	}
	return e
}

func (e *Embed) WithField(name, value string, inline bool) *Embed {
	e.Fields = append(e.Fields, EmbedField{
		name,
		value,
		inline,
	})

	return e
}

type EmbedFooter struct {
	Text    string `json:"footer"`
	IconUrl string `json:"icon_url"`
}

type EmbedImage struct {
	Url string `json:"url"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	IconUrl string `json:"icon_url"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// very very very safe function, 100% wont break if wrong format is passed in :)
func HexToDecimal(hex string) int64 {
	hex = strings.TrimPrefix(hex, "#")
	val, _ := strconv.ParseInt(hex, 16, 64)
	return val
}

func SendWebhookPayload(payload *WebhookPayload, url string) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	return resp, nil
}
