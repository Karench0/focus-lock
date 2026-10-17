package kotlinquiz

import (
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// This source-only fixture is an oracle for API and browser integration tests.
// It is never embedded in the application or copied into release packages.
// Regenerate after editing the bank with FOCUS_LOCK_REGENERATE_QUIZ_ORACLE=1.
func TestIntegrationOracleMatchesBank(t *testing.T) {
	path := filepath.Join("..", "..", "tests", "oracles", "kotlin-quiz.json")
	expected := make(map[string]string, len(pool))
	for _, item := range pool {
		expected[item.prompt] = item.options[item.correct]
	}
	if os.Getenv("FOCUS_LOCK_REGENERATE_QUIZ_ORACLE") == "1" {
		encoded, err := json.MarshalIndent(expected, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var actual map[string]string
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("integration oracle differs from private bank; regenerate with FOCUS_LOCK_REGENERATE_QUIZ_ORACLE=1")
	}
}

func TestPoolStructure(t *testing.T) {
	if len(pool) != 20 {
		t.Fatalf("got %d questions; want 20", len(pool))
	}
	prompts := make(map[string]bool)
	for i, item := range pool {
		if strings.TrimSpace(item.prompt) == "" || prompts[item.prompt] {
			t.Errorf("question %d has empty or repeated prompt", i)
		}
		prompts[item.prompt] = true
		if len(item.options) != 8 || item.correct < 0 || item.correct >= len(item.options) {
			t.Fatalf("question %d has invalid answer layout", i)
		}
		options := make(map[string]bool)
		for _, option := range item.options {
			normalized := strings.ToLower(strings.TrimSpace(option))
			if normalized == "" || options[normalized] {
				t.Errorf("question %d has empty or duplicate option %q", i, option)
			}
			options[normalized] = true
		}
		source, err := url.Parse(item.sourceURL)
		if err != nil || source.Scheme != "https" || source.Hostname() != "kotlinlang.org" || source.Path == "" {
			t.Errorf("question %d needs a specific official HTTPS source: %q", i, item.sourceURL)
		}
	}
}

func TestQuestionJSONContainsOnlyPublicChallenge(t *testing.T) {
	q := New()
	question, err := q.Get("session")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(question)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Fatalf("unexpected serialized fields: %s", encoded)
	}
	for _, key := range []string{"token", "prompt", "options", "sourceUrl"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("missing field %s", key)
		}
	}
	var privateState map[string]json.RawMessage
	stateJSON, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stateJSON, &privateState); err != nil || len(privateState) != 0 {
		t.Fatalf("Quiz must not serialize its private answers: %s (%v)", stateJSON, err)
	}
	decoded, err := hex.DecodeString(question.Token)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("token is not 256 random bits: %q", question.Token)
	}
}

func TestPendingQuestionCannotBeRerolledOrMutated(t *testing.T) {
	var q Quiz // The zero value also works.
	first, err := q.Get("session")
	if err != nil {
		t.Fatal(err)
	}
	original := cloneQuestion(first)
	first.Options[0] = "mutated client copy"
	first.Prompt = "mutated prompt"
	for range 10 {
		again, err := q.Get("session")
		if err != nil || !reflect.DeepEqual(again, original) {
			t.Fatalf("pending challenge changed: %#v (%v)", again, err)
		}
	}
	other, err := q.Get("other-session")
	if err != nil || other.Token == original.Token {
		t.Fatalf("sessions must have independent tokens: %#v (%v)", other, err)
	}
	again, err := q.Get("session")
	if err != nil || !reflect.DeepEqual(again, original) {
		t.Fatalf("another session changed the existing challenge: %#v (%v)", again, err)
	}
}

func TestRejectInvalidInputWithoutConsumingChallenge(t *testing.T) {
	q := New()
	if _, err := q.Get(""); err == nil {
		t.Fatal("empty session was accepted")
	}
	question, err := q.Get("session")
	if err != nil {
		t.Fatal(err)
	}
	other, err := q.Get("other-session")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		session string
		token   string
		option  int
	}{
		{"empty session", "", question.Token, 0},
		{"unknown session", "missing", question.Token, 0},
		{"foreign session", "other-session", question.Token, 0},
		{"foreign token", "session", other.Token, 0},
		{"empty token", "session", "", 0},
		{"altered token", "session", question.Token + "0", 0},
		{"negative option", "session", question.Token, -1},
		{"out of range option", "session", question.Token, 8},
	} {
		t.Run(test.name, func(t *testing.T) {
			if correct, err := q.Answer(test.session, test.token, test.option); err == nil || correct {
				t.Fatalf("invalid input accepted: %v %v", correct, err)
			}
			again, err := q.Get("session")
			if err != nil || !reflect.DeepEqual(question, again) {
				t.Fatalf("invalid input consumed question: %#v %v", again, err)
			}
		})
	}
	if correct, err := q.Answer("session", question.Token, q.sessions["session"].pending.correct); err != nil || !correct {
		t.Fatalf("valid answer rejected after invalid attempts: %v %v", correct, err)
	}
}

func TestConsumeAnswersAndRejectReplay(t *testing.T) {
	for _, shouldBeCorrect := range []bool{false, true} {
		q := New()
		question, err := q.Get("session")
		if err != nil {
			t.Fatal(err)
		}
		option := q.sessions["session"].pending.correct
		if !shouldBeCorrect {
			option = (option + 1) % 8
		}
		correct, err := q.Answer("session", question.Token, option)
		if err != nil || correct != shouldBeCorrect {
			t.Fatalf("answer mismatch: got %v %v, want %v", correct, err, shouldBeCorrect)
		}
		if correct, err := q.Answer("session", question.Token, option); err == nil || correct {
			t.Fatal("consumed token was replayed")
		}
		next, err := q.Get("session")
		if err != nil {
			t.Fatal(err)
		}
		if next.Token == question.Token || next.Prompt == question.Prompt {
			t.Fatal("next challenge reused the token or previous question")
		}
		if correct, err := q.Answer("session", question.Token, option); err == nil || correct {
			t.Fatal("old token was accepted for a newer challenge")
		}
		again, err := q.Get("session")
		if err != nil || !reflect.DeepEqual(next, again) {
			t.Fatal("replay consumed the newer challenge")
		}
	}
}

func TestResetInvalidatesAllSessions(t *testing.T) {
	q := New()
	old := make(map[string]Question)
	for _, session := range []string{"one", "two"} {
		question, err := q.Get(session)
		if err != nil {
			t.Fatal(err)
		}
		old[session] = question
	}
	q.Reset()
	if len(q.sessions) != 0 {
		t.Fatal("Reset retained session state")
	}
	for session, question := range old {
		if correct, err := q.Answer(session, question.Token, 0); err == nil || correct {
			t.Fatal("Reset retained a usable token")
		}
		fresh, err := q.Get(session)
		if err != nil || fresh.Token == question.Token {
			t.Fatalf("fresh session token missing after Reset: %v", err)
		}
	}
}

func TestEveryQuestionAndShuffledAnswerCanBeUsed(t *testing.T) {
	for i, item := range pool {
		positions := [8]bool{}
		for attempt := range 512 {
			pending, err := makeChallenge(item)
			if err != nil {
				t.Fatal(err)
			}
			if pending.question.Options[pending.correct] != item.options[item.correct] {
				t.Fatalf("question %d lost its answer key during shuffle", i)
			}
			positions[pending.correct] = true
			// Check all eight submitted choices against the private key; each
			// valid attempt gets its own one-use challenge state.
			option := attempt % 8
			q := &Quiz{sessions: map[string]*sessionState{"session": {pending: pending, last: i}}}
			correct, err := q.Answer("session", pending.question.Token, option)
			if err != nil || correct != (option == pending.correct) {
				t.Fatalf("question %d: shuffled option %d mismatch: %v %v", i, option, correct, err)
			}
		}
		for position, seen := range positions {
			if !seen {
				t.Errorf("question %d never shuffled its answer to position %d in 512 attempts", i, position)
			}
		}
	}
}

func TestRandomSelectionReachesEntirePoolWithoutImmediateRepeats(t *testing.T) {
	q := New()
	seen := make(map[string]bool)
	previous := ""
	for range 2048 {
		question, err := q.Get("session")
		if err != nil {
			t.Fatal(err)
		}
		if question.Prompt == previous {
			t.Fatal("immediately repeated a question")
		}
		seen[question.Prompt] = true
		previous = question.Prompt
		wrong := (q.sessions["session"].pending.correct + 1) % 8
		if correct, err := q.Answer("session", question.Token, wrong); err != nil || correct {
			t.Fatalf("wrong answer accepted: %v %v", correct, err)
		}
	}
	if len(seen) != len(pool) {
		t.Fatalf("selection only reached %d of %d questions", len(seen), len(pool))
	}
}
