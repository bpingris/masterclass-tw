package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/joho/godotenv"
)

func stream(listenTo string) error {
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
	res, err := twitwi.SetRules(ctx, []ValueTag{{Value: fmt.Sprintf("from:%s", listenTo), Tag: fmt.Sprintf("from %s", listenTo)}})
	if err != nil {
		return fmt.Errorf("set rules: %w", err)
	}
	res, err = twitwi.Stream(ctx)
	if err != nil {
		return fmt.Errorf("stream: %w", err)
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
	for {
		var m StreamResponse
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode response: %w", err)
		}
		log.Printf("found a tweet: %s - %s\n", m.Data.Text, m.Data.ID)
		twitwi.Send(ctx, WithReplyID(m.Data.ID), WithText("masterclass"))
	}
	return nil
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return err
	}
	listenTo := flag.String("account", "", "the account to listen to its tweets")
	flag.Parse()
	if *listenTo == "" {
		flag.Usage()
		return fmt.Errorf("invalid `listenTo` flag")
	}
	return stream(*listenTo)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
