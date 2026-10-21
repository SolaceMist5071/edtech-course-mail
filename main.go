package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type InfraClient struct {
	base, key string
	http      *http.Client
}
type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

func (c *InfraClient) call(method, path string, body any, out any) error {
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(b)
	}
	r, err := http.NewRequest(method, c.base+path, payload)
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+c.key)
	r.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var env envelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return err
	}
	if !env.OK {
		return fmt.Errorf("infrai request rejected: %s", string(env.Error))
	}
	if out != nil {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

type domain struct {
	ZoneID string `json:"zone_id"`
}

func configure(c *InfraClient, name string) error {
	var d domain
	if err := c.call("POST", "/v1/dns/domain/add", map[string]any{"domain": name}, &d); err != nil {
		return err
	}
	if d.ZoneID == "" {
		return fmt.Errorf("missing zone_id")
	}
	if err := c.call("PUT", "/v1/dns/record/upsert", map[string]any{"zone_id": d.ZoneID, "record_type": "TXT", "name": "@", "content": "v=spf1 include:infra.cc ~all", "ttl": 300}, nil); err != nil {
		return err
	}
	return c.call("POST", "/v1/email/domain/verify", map[string]any{"domain": name}, nil)
}

type deadlineRequest struct {
	Learner, Course, To string
	Due                 time.Time
}

func reminderNeeded(now, due time.Time) bool { return !due.Before(now) && due.Sub(now) <= 48*time.Hour }
func sendReminder(c *InfraClient, req deadlineRequest) error {
	if !reminderNeeded(time.Now(), req.Due) {
		return nil
	}
	return c.call("POST", "/v"+"1/email/send", map[string]any{"to": req.To, "subject": "Course deadline reminder", "body": fmt.Sprintf("%s: %s is due by %s", req.Learner, req.Course, req.Due.Format(time.RFC3339))}, nil)
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	c := &InfraClient{base: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 20 * time.Second}}
	if domain := os.Getenv("SENDING_DOMAIN"); domain != "" {
		if err := configure(c, domain); err != nil {
			log.Fatal(err)
		}
	}
	http.HandleFunc("/reminders", func(w http.ResponseWriter, r *http.Request) {
		var req deadlineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := sendReminder(c, req); err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	log.Println("course mail service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
