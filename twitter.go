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
	Tag   string `json:"tag,omitempty"`
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
	r.Add("Authorization", "Bearer "+t.Bearer)
	r.Add("User-Agent", "v2TweetLookupPython")
	r.Add("Content-Type", "application/json")
}

type QueryParam struct {
	key   string
	value string
}

func (t *Twitwi) request(ctx context.Context, method string, url string, body io.Reader, params ...QueryParam) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("prepare request: %w", err)
	}
	t.setHeader(&req.Header)
	if len(params) > 0 {
		q := req.URL.Query()
		for _, param := range params {
			q.Add(param.key, param.value)
		}
		req.URL.RawQuery = q.Encode()
	}
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

func (t *Twitwi) Status(ctx context.Context, id string) (*TweetStatus, error) {
	res, err := t.request(ctx, http.MethodGet, "https://api.twitter.com/1.1/statuses/show.json?id="+id, nil)
	if err != nil {
		return nil, err
	}
	var ts TweetStatus
	if err = json.NewDecoder(res.Body).Decode(&ts); err != nil {
		return nil, err
	}
	return &ts, nil
}

func (t *Twitwi) Stream(ctx context.Context, params ...QueryParam) (*http.Response, error) {
	return t.request(ctx, http.MethodGet, "https://api.twitter.com/2/tweets/search/stream",
		nil, params...)
}

type TweetStatus struct {
	CreatedAt string `json:"created_at"`
	ID        int64  `json:"id"`
	IDStr     string `json:"id_str"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
	Entities  struct {
		Hashtags     []interface{} `json:"hashtags"`
		Symbols      []interface{} `json:"symbols"`
		UserMentions []interface{} `json:"user_mentions"`
		Urls         []interface{} `json:"urls"`
		Media        []struct {
			ID            int64  `json:"id"`
			IDStr         string `json:"id_str"`
			Indices       []int  `json:"indices"`
			MediaURL      string `json:"media_url"`
			MediaURLHTTPS string `json:"media_url_https"`
			URL           string `json:"url"`
			DisplayURL    string `json:"display_url"`
			ExpandedURL   string `json:"expanded_url"`
			Type          string `json:"type"`
			Sizes         struct {
				Small struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"small"`
				Thumb struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"thumb"`
				Medium struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"medium"`
				Large struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"large"`
			} `json:"sizes"`
			SourceStatusID    int64  `json:"source_status_id"`
			SourceStatusIDStr string `json:"source_status_id_str"`
			SourceUserID      int64  `json:"source_user_id"`
			SourceUserIDStr   string `json:"source_user_id_str"`
		} `json:"media"`
	} `json:"entities"`
	ExtendedEntities struct {
		Media []struct {
			ID            int64  `json:"id"`
			IDStr         string `json:"id_str"`
			Indices       []int  `json:"indices"`
			MediaURL      string `json:"media_url"`
			MediaURLHTTPS string `json:"media_url_https"`
			URL           string `json:"url"`
			DisplayURL    string `json:"display_url"`
			ExpandedURL   string `json:"expanded_url"`
			Type          string `json:"type"`
			Sizes         struct {
				Small struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"small"`
				Thumb struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"thumb"`
				Medium struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"medium"`
				Large struct {
					W      int    `json:"w"`
					H      int    `json:"h"`
					Resize string `json:"resize"`
				} `json:"large"`
			} `json:"sizes"`
			SourceStatusID    int64  `json:"source_status_id"`
			SourceStatusIDStr string `json:"source_status_id_str"`
			SourceUserID      int64  `json:"source_user_id"`
			SourceUserIDStr   string `json:"source_user_id_str"`
			VideoInfo         struct {
				AspectRatio    []int `json:"aspect_ratio"`
				DurationMillis int   `json:"duration_millis"`
				Variants       []struct {
					Bitrate     int    `json:"bitrate,omitempty"`
					ContentType string `json:"content_type"`
					URL         string `json:"url"`
				} `json:"variants"`
			} `json:"video_info"`
			AdditionalMediaInfo struct {
				Monetizable bool `json:"monetizable"`
				SourceUser  struct {
					ID          int64       `json:"id"`
					IDStr       string      `json:"id_str"`
					Name        string      `json:"name"`
					ScreenName  string      `json:"screen_name"`
					Location    string      `json:"location"`
					Description string      `json:"description"`
					URL         interface{} `json:"url"`
					Entities    struct {
						Description struct {
							Urls []interface{} `json:"urls"`
						} `json:"description"`
					} `json:"entities"`
					Protected                      bool          `json:"protected"`
					FollowersCount                 int           `json:"followers_count"`
					FriendsCount                   int           `json:"friends_count"`
					ListedCount                    int           `json:"listed_count"`
					CreatedAt                      string        `json:"created_at"`
					FavouritesCount                int           `json:"favourites_count"`
					UtcOffset                      interface{}   `json:"utc_offset"`
					TimeZone                       interface{}   `json:"time_zone"`
					GeoEnabled                     bool          `json:"geo_enabled"`
					Verified                       bool          `json:"verified"`
					StatusesCount                  int           `json:"statuses_count"`
					Lang                           interface{}   `json:"lang"`
					ContributorsEnabled            bool          `json:"contributors_enabled"`
					IsTranslator                   bool          `json:"is_translator"`
					IsTranslationEnabled           bool          `json:"is_translation_enabled"`
					ProfileBackgroundColor         string        `json:"profile_background_color"`
					ProfileBackgroundImageURL      interface{}   `json:"profile_background_image_url"`
					ProfileBackgroundImageURLHTTPS interface{}   `json:"profile_background_image_url_https"`
					ProfileBackgroundTile          bool          `json:"profile_background_tile"`
					ProfileImageURL                string        `json:"profile_image_url"`
					ProfileImageURLHTTPS           string        `json:"profile_image_url_https"`
					ProfileBannerURL               string        `json:"profile_banner_url"`
					ProfileLinkColor               string        `json:"profile_link_color"`
					ProfileSidebarBorderColor      string        `json:"profile_sidebar_border_color"`
					ProfileSidebarFillColor        string        `json:"profile_sidebar_fill_color"`
					ProfileTextColor               string        `json:"profile_text_color"`
					ProfileUseBackgroundImage      bool          `json:"profile_use_background_image"`
					HasExtendedProfile             bool          `json:"has_extended_profile"`
					DefaultProfile                 bool          `json:"default_profile"`
					DefaultProfileImage            bool          `json:"default_profile_image"`
					Following                      interface{}   `json:"following"`
					FollowRequestSent              interface{}   `json:"follow_request_sent"`
					Notifications                  interface{}   `json:"notifications"`
					TranslatorType                 string        `json:"translator_type"`
					WithheldInCountries            []interface{} `json:"withheld_in_countries"`
				} `json:"source_user"`
			} `json:"additional_media_info"`
		} `json:"media"`
	} `json:"extended_entities"`
	Source               string      `json:"source"`
	InReplyToStatusID    interface{} `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr interface{} `json:"in_reply_to_status_id_str"`
	InReplyToUserID      interface{} `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   interface{} `json:"in_reply_to_user_id_str"`
	InReplyToScreenName  interface{} `json:"in_reply_to_screen_name"`
	User                 struct {
		ID          int64       `json:"id"`
		IDStr       string      `json:"id_str"`
		Name        string      `json:"name"`
		ScreenName  string      `json:"screen_name"`
		Location    string      `json:"location"`
		Description string      `json:"description"`
		URL         interface{} `json:"url"`
		Entities    struct {
			Description struct {
				Urls []interface{} `json:"urls"`
			} `json:"description"`
		} `json:"entities"`
		Protected                      bool          `json:"protected"`
		FollowersCount                 int           `json:"followers_count"`
		FriendsCount                   int           `json:"friends_count"`
		ListedCount                    int           `json:"listed_count"`
		CreatedAt                      string        `json:"created_at"`
		FavouritesCount                int           `json:"favourites_count"`
		UtcOffset                      interface{}   `json:"utc_offset"`
		TimeZone                       interface{}   `json:"time_zone"`
		GeoEnabled                     bool          `json:"geo_enabled"`
		Verified                       bool          `json:"verified"`
		StatusesCount                  int           `json:"statuses_count"`
		Lang                           interface{}   `json:"lang"`
		ContributorsEnabled            bool          `json:"contributors_enabled"`
		IsTranslator                   bool          `json:"is_translator"`
		IsTranslationEnabled           bool          `json:"is_translation_enabled"`
		ProfileBackgroundColor         string        `json:"profile_background_color"`
		ProfileBackgroundImageURL      interface{}   `json:"profile_background_image_url"`
		ProfileBackgroundImageURLHTTPS interface{}   `json:"profile_background_image_url_https"`
		ProfileBackgroundTile          bool          `json:"profile_background_tile"`
		ProfileImageURL                string        `json:"profile_image_url"`
		ProfileImageURLHTTPS           string        `json:"profile_image_url_https"`
		ProfileBannerURL               string        `json:"profile_banner_url"`
		ProfileLinkColor               string        `json:"profile_link_color"`
		ProfileSidebarBorderColor      string        `json:"profile_sidebar_border_color"`
		ProfileSidebarFillColor        string        `json:"profile_sidebar_fill_color"`
		ProfileTextColor               string        `json:"profile_text_color"`
		ProfileUseBackgroundImage      bool          `json:"profile_use_background_image"`
		HasExtendedProfile             bool          `json:"has_extended_profile"`
		DefaultProfile                 bool          `json:"default_profile"`
		DefaultProfileImage            bool          `json:"default_profile_image"`
		Following                      interface{}   `json:"following"`
		FollowRequestSent              interface{}   `json:"follow_request_sent"`
		Notifications                  interface{}   `json:"notifications"`
		TranslatorType                 string        `json:"translator_type"`
		WithheldInCountries            []interface{} `json:"withheld_in_countries"`
	} `json:"user"`
	Geo                         interface{} `json:"geo"`
	Coordinates                 interface{} `json:"coordinates"`
	Place                       interface{} `json:"place"`
	Contributors                interface{} `json:"contributors"`
	IsQuoteStatus               bool        `json:"is_quote_status"`
	RetweetCount                int         `json:"retweet_count"`
	FavoriteCount               int         `json:"favorite_count"`
	Favorited                   bool        `json:"favorited"`
	Retweeted                   bool        `json:"retweeted"`
	PossiblySensitive           bool        `json:"possibly_sensitive"`
	PossiblySensitiveAppealable bool        `json:"possibly_sensitive_appealable"`
	Lang                        string      `json:"lang"`
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
