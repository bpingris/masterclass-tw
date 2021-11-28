package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

func stream() error {
	ctx := context.Background()
	twitwi := NewTwi()
	rules, err := twitwi.GetRules(ctx)
	if err != nil {
		return fmt.Errorf("get rules: %w", err)
	}
	_, err = twitwi.DeleteAllRules(ctx, rules)
	if err != nil {
		return fmt.Errorf("delete all rules: %w", err)
	}
	res, err := twitwi.SetRules(ctx, []ValueTag{{Value: "@PlsSaveThis"}})
	if err != nil {
		return fmt.Errorf("set rules: %w", err)
	}
	res, err = twitwi.Stream(ctx, []QueryParam{
		{"expansions", "author_id,referenced_tweets.id"},
		{"user.fields", "username"},
		{"tweet.fields", "in_reply_to_user_id"},
	}...)
	if err != nil {
		return fmt.Errorf("stream: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status: %d", res.StatusCode)
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)

	log.Println("stream started")
	for {
		var sr StreamResponse
		if err := dec.Decode(&sr); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode response: %w", err)
		}

		if sr.Data.AuthorID == "3401628065" {
			log.Println("Ignoring own tweets")
			continue
		}

		log.Printf("found a tweet: %s - %s - %s\n", sr.Data.Text, sr.Data.ID, sr.Data.AuthorID)
		go handle(twitwi, &sr)
	}
	return nil
}

type StreamResponse struct {
	Data struct {
		ID               string `json:"id"`
		Text             string `json:"text"`
		AuthorID         string `json:"author_id"`
		InReplyToUserID  string `json:"in_reply_to_user_id"`
		ReferencedTweets []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"referenced_tweets"`
	} `json:"data"`
	Includes struct {
		Tweets []struct {
			AuthorID string `json:"author_id"`
			ID       string `json:"id"`
			Text     string `json:"text"`
		} `json:"tweets"`
		Users []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
		}
	} `json:"includes"`
	MatchingRules []struct {
		ID  string `json:"id"`
		Tag string `json:"tag"`
	} `json:"matching_rules"`
}

func handle(t *Twitwi, sr *StreamResponse) {
	if len(sr.Includes.Tweets) != 1 {
		log.Printf("tweets: %d\n", len(sr.Includes.Tweets))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	ts, err := t.Status(ctx, sr.Includes.Tweets[0].ID)
	if err != nil {
		log.Println(err)
		return
	}
	if len(ts.ExtendedEntities.Media) == 0 {
		return
	}
	res, err := t.Send(ctx, WithReplyID(sr.Data.ID), WithText(ts.ExtendedEntities.Media[0].VideoInfo.Variants[0].URL))
	if err != nil {
		log.Printf("send tweet: %v", err)
		return
	}
	if res.StatusCode != http.StatusOK {
		log.Printf("invalid status code: %d", res.StatusCode)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	return stream()
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
