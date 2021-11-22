package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

func stream(account string) error {
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
	res, err := twitwi.SetRules(ctx, []ValueTag{{Value: "from:" + account, Tag: "from " + account}})
	if err != nil {
		return fmt.Errorf("set rules: %w", err)
	}
	res, err = twitwi.Stream(ctx)
	if err != nil {
		return fmt.Errorf("stream: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status: %d", res.StatusCode)
	}
	defer res.Body.Close()

	dec := json.NewDecoder(res.Body)

	type StreamResponse struct {
		Data struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"data"`
		MatchingRules []struct {
			ID  string `json:"id"`
			Tag string `json:"tag"`
		} `json:"matching_rules"`
	}
	log.Println("stream started")
	for {
		var m StreamResponse
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode response: %w", err)
		}
		log.Printf("found a tweet: %s - %s\n", m.Data.Text, m.Data.ID)
		res, err := twitwi.Send(ctx, WithReplyID(m.Data.ID), WithText("masterclass"))
		if err != nil {
			return errors.Wrap(err, "send tweet")
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusCreated {
			b, err := io.ReadAll(res.Body)
			if err != nil {
				return errors.Wrap(err, "read body")
			}
			log.Printf("%s", string(b))
			return fmt.Errorf("invalid status: %d", res.StatusCode)
		}
	}
	return nil
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	account := flag.String("account", "", "the account to listen to its tweets")
	flag.Parse()
	if *account == "" {
		flag.Usage()
		return fmt.Errorf("invalid `account` flag")
	}
	return stream(*account)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
