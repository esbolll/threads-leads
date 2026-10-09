package main

import "testing"

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
	for _, c := range cases {
		lead, ok := classify(Post{Text: c.text})
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
		t.Error("need should not match inside needle")
	}
	if !wordMatch("ищем devops-инженера", "devops") {
		t.Error("devops should match before hyphen as boundary")
	}
}
