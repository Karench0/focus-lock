// Package kotlinquiz keeps answer keys and one-use challenges in the Go service.
// Callers serialize access; Quiz deliberately has no internal locking.
package kotlinquiz

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
)

// Question is the complete client-visible challenge. It never contains an answer key.
type Question struct {
	Token     string   `json:"token"`
	Prompt    string   `json:"prompt"`
	Options   []string `json:"options"`
	SourceURL string   `json:"sourceUrl"`
}

type challenge struct {
	question Question
	correct  int
}

type sessionState struct {
	pending *challenge
	last    int
}

type Quiz struct {
	sessions map[string]*sessionState
}

func New() *Quiz {
	return &Quiz{sessions: make(map[string]*sessionState)}
}

// Get preserves an unanswered challenge, including its option order and token.
func (q *Quiz) Get(sessionID string) (Question, error) {
	if sessionID == "" {
		return Question{}, errors.New("для вопроса нужна активная сессия")
	}
	state := q.sessions[sessionID]
	if state != nil && state.pending != nil {
		return cloneQuestion(state.pending.question), nil
	}
	last := -1
	if state != nil {
		last = state.last
	}
	count := len(pool)
	if last >= 0 {
		count--
	}
	index, err := randomIndex(count)
	if err != nil {
		return Question{}, err
	}
	if last >= 0 && index >= last {
		index++
	}
	pending, err := makeChallenge(pool[index])
	if err != nil {
		return Question{}, err
	}
	if q.sessions == nil {
		q.sessions = make(map[string]*sessionState)
	}
	q.sessions[sessionID] = &sessionState{pending: pending, last: index}
	return cloneQuestion(pending.question), nil
}

// Answer consumes every valid submission, whether correct or incorrect. Invalid
// indices, foreign tokens, and replays do not consume the current challenge.
func (q *Quiz) Answer(sessionID, token string, option int) (bool, error) {
	state := q.sessions[sessionID]
	if state == nil || state.pending == nil || token == "" ||
		subtle.ConstantTimeCompare([]byte(token), []byte(state.pending.question.Token)) != 1 {
		return false, errors.New("вопрос устарел или не относится к этой сессии")
	}
	if option < 0 || option >= len(state.pending.question.Options) {
		return false, errors.New("выберите один из предложенных вариантов")
	}
	correct := option == state.pending.correct
	state.pending = nil
	return correct, nil
}

func (q *Quiz) Reset() {
	q.sessions = make(map[string]*sessionState)
}

func randomIndex(n int) (int, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func makeChallenge(item entry) (*challenge, error) {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, err
	}
	order := [8]int{0, 1, 2, 3, 4, 5, 6, 7}
	for i := len(order) - 1; i > 0; i-- {
		j, err := randomIndex(i + 1)
		if err != nil {
			return nil, err
		}
		order[i], order[j] = order[j], order[i]
	}
	result := &challenge{question: Question{
		Token: hex.EncodeToString(token[:]), Prompt: item.prompt,
		Options: make([]string, len(order)), SourceURL: item.sourceURL,
	}}
	for i, original := range order {
		result.question.Options[i] = item.options[original]
		if original == item.correct {
			result.correct = i
		}
	}
	return result, nil
}

func cloneQuestion(question Question) Question {
	question.Options = append([]string(nil), question.Options...)
	return question
}
