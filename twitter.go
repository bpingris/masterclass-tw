package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/oauth1"
)

type Twitwi struct {
	Bearer            string
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type Rules struct {
	Data []struct {
		ID    string `json:"id"`
		Value string `json:"value"`
		Tag   string `json:"tag"`
	} `json:"data"`
	Meta struct {
		Sent        time.Time `json:"sent"`
		ResultCount int       `json:"result_count"`
	} `json:"meta"`
}

type ValueTag struct {
	Value string `json:"value"`
	Tag   string `json:"tag"`
}

type AddRules struct {
	Add []ValueTag `json:"add"`
}

func NewTwi() *Twitwi {
	return &Twitwi{
		Bearer:            os.Getenv("TWITWI_BEARER"),
		ConsumerKey:       os.Getenv("TWITWI_CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("TWITWI_CONSUMER_SECRET"),
		AccessToken:       os.Getenv("TWITWI_ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("TWITWI_ACCESS_TOKEN_SECRET"),
	}
}

func (t *Twitwi) getAuthClient() *http.Client {
	cfg := oauth1.Config{
		ConsumerKey:    t.ConsumerKey,
		ConsumerSecret: t.ConsumerSecret,
	}
	client := oauth1.NewClient(context.Background(), &cfg, oauth1.NewToken(t.AccessToken, t.AccessTokenSecret))
	return client
}

func (t *Twitwi) setHeader(r *http.Header) {
	r.Add("Authorization", fmt.Sprintf("Bearer %s", t.Bearer))
	r.Add("User-Agent", "v2TweetLookupPython")
	r.Add("Content-Type", "application/json")
}

func (t *Twitwi) request(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("prepare request: %w", err)
	}
	t.setHeader(&req.Header)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	return res, nil
}

func (t *Twitwi) GetRules(ctx context.Context) (*Rules, error) {
	res, err := t.request(ctx, http.MethodGet, "https://api.twitter.com/2/tweets/search/stream/rules", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var rules Rules
	if err = json.NewDecoder(res.Body).Decode(&rules); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &rules, nil
}

func (t *Twitwi) DeleteAllRules(ctx context.Context, rules *Rules) (*http.Response, error) {
	if len(rules.Data) == 0 {
		return nil, nil
	}
	type payload struct {
		Delete struct {
			IDs []string `json:"ids"`
		} `json:"delete"`
	}
	ids := make([]string, len(rules.Data))
	for i := range rules.Data {
		ids[i] = rules.Data[i].ID
	}
	pl, err := json.Marshal(payload{Delete: struct {
		IDs []string `json:"ids"`
	}{IDs: ids}})
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return t.request(ctx, http.MethodPost, "https://api.twitter.com/2/tweets/search/stream/rules", bytes.NewBuffer(pl))
}

func (t *Twitwi) SetRules(ctx context.Context, rules []ValueTag) (*http.Response, error) {
	pl, err := json.Marshal(AddRules{Add: rules})
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return t.request(ctx, http.MethodPost, "https://api.twitter.com/2/tweets/search/stream/rules", bytes.NewBuffer(pl))
}

func (t *Twitwi) Stream(ctx context.Context) (*http.Response, error) {
	return t.request(ctx, http.MethodGet, "https://api.twitter.com/2/tweets/search/stream", nil)
}

type SendTweetPayload struct {
	Text                  string `json:"text,omitempty"`
	DirectMessageDeepLink string `json:"direct_message_deep_link,omitempty"`
	ForSuperFollowersOnly bool   `json:"for_super_followers_only,omitempty"`
	Geo                   *struct {
		PlaceID string `json:"place_id,omitempty"`
	} `json:"geo,omitempty"`
	Media *struct {
		MediaIDs     []string `json:"media_ids,omitempty"`
		TaggedUserID []string `json:"tagged_user_id,omitempty"`
	} `json:"media,omitempty"`
	Poll *struct {
		DurationMinutes int      `json:"duration_minutes,omitempty"`
		Options         []string `json:"options,omitempty"`
	} `json:"poll,omitempty"`
	QuoteTweetID string `json:"quote_tweet_id,omitempty"`
	Reply        *struct {
		ExcludeReplyUserIds []string `json:"exclude_reply_user_ids,omitempty"`
		InReplyToTweetID    string   `json:"in_reply_to_tweet_id,omitempty"`
	} `json:"reply,omitempty"`
	ReplySettings string `json:"reply_settings,omitempty"`
}

type Option func(*SendTweetPayload)

func WithText(text string) Option {
	return func(stp *SendTweetPayload) {
		stp.Text = text
	}
}

func WithReplyID(id string) Option {
	return func(stp *SendTweetPayload) {
		if stp.Reply == nil {
			stp.Reply = &struct {
				ExcludeReplyUserIds []string `json:"exclude_reply_user_ids,omitempty"`
				InReplyToTweetID    string   `json:"in_reply_to_tweet_id,omitempty"`
			}{}
		}
		stp.Reply.InReplyToTweetID = id
	}
}

func (t *Twitwi) Send(ctx context.Context, opts ...Option) (*http.Response, error) {
	sendTweetPayload := SendTweetPayload{}
	for _, opt := range opts {
		opt(&sendTweetPayload)
	}
	b, err := json.Marshal(sendTweetPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	client := t.getAuthClient()
	req, err := http.NewRequest(http.MethodPost, "https://api.twitter.com/2/tweets", bytes.NewBuffer(b))
	req.Header.Add("content-type", "application/json")

	return client.Do(req)
}
