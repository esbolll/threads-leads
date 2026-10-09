package filter

import (
	"testing"
	"time"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		text string
		want bool
		cat  string
	}{
		{"Ищу DevOps для настройки Kubernetes и CI/CD.", true, "unknown"},
		{"Looking for a DevOps engineer for a 2 month AWS migration project.", true, "project"},
		{"Нужен Python разработчик на небольшой проект.", true, "project"},
		{"Сегодня изучал Kubernetes.", false, ""},
		{"DevOps is dead.", false, ""},
		{"5 лучших инструментов для разработчика.", false, ""},
		{"Ищу работу devops, рассмотрю предложения", false, ""},
		{"I need a needle, not a developer", true, "unknown"},
	}
	now := time.Now()
	for _, c := range cases {
		lead, ok := Classify(model.Post{Text: c.text}, now)
		if ok != c.want {
			t.Errorf("%q: relevant=%v want %v (tech=%v intent=%v)", c.text, ok, c.want, lead.MatchedTech, lead.MatchedIntent)
			continue
		}
		if ok && lead.Category != c.cat {
			t.Errorf("%q: category=%s want %s", c.text, lead.Category, c.cat)
		}
	}
}

func TestWordMatch(t *testing.T) {
	if wordMatch("needle", "need") {
		t.Error("need must not match inside needle")
	}
	if !wordMatch("ищем devops-инженера", "devops") {
		t.Error("hyphen must act as a word boundary")
	}
}

func TestPostKey(t *testing.T) {
	a := model.Post{Permalink: "https://www.threads.com/@u/post/abc?x=1"}
	b := model.Post{Permalink: "https://www.threads.com/@u/post/abc/"}
	if a.Key() != b.Key() {
		t.Errorf("keys differ: %s vs %s", a.Key(), b.Key())
	}
	c := model.Post{Username: "u", Text: "hi"}
	if c.Key() == "" || c.Key()[:5] != "hash:" {
		t.Errorf("hash key expected, got %s", c.Key())
	}
}
