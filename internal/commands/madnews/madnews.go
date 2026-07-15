package madnews

import (
	_ "embed"
	"encoding/json"
	"math/rand"
	"regexp"
	"strings"
	"sync"
)

//go:embed data/ru.json
var ruDictionaryJSON []byte

type predict struct {
	Message string `json:"message"`
	Sex     string `json:"sex"`
}

type replacementSet struct {
	Key          string   `json:"key"`
	Replacements []string `json:"replacements"`
}

type dictionary struct {
	Predicts    []predict                 `json:"predicts"`
	Actions     map[string][]string       `json:"actions"`
	Conclusions map[string][]string       `json:"conclusions"`
	Sets        []replacementSet          `json:"sets"`
}

type generator struct {
	dict        dictionary
	sex         string
	predicts    []predict
	actions     map[string][]string
	conclusions map[string][]string
}

var (
	ruMu sync.Mutex
	ruGen *generator
)

var placeholderRE = regexp.MustCompile(`\[[\p{Cyrillic}\w]*\]`)

func Generate(lang string) (string, error) {
	if lang == "" {
		lang = "ru"
	}
	if lang != "ru" {
		lang = "ru"
	}
	ruMu.Lock()
	defer ruMu.Unlock()
	if ruGen == nil {
		gen, err := newGenerator(ruDictionaryJSON)
		if err != nil {
			return "", err
		}
		ruGen = gen
	}
	return ruGen.generate(), nil
}

func newGenerator(raw []byte) (*generator, error) {
	var dict dictionary
	if err := json.Unmarshal(raw, &dict); err != nil {
		return nil, err
	}
	return &generator{
		dict:        dict,
		predicts:    clonePredicts(dict.Predicts),
		actions:     cloneActions(dict.Actions),
		conclusions: cloneConclusions(dict.Conclusions),
	}, nil
}

func (g *generator) generate() string {
	person := strings.TrimSpace(g.getPerson())
	action := strings.TrimSpace(g.getAction())
	conclusion := strings.TrimSpace(g.getConclusion())
	text := person + " " + action + " " + conclusion
	return strings.TrimSpace(strings.ReplaceAll(text, "  ", " "))
}

func (g *generator) getPerson() string {
	if len(g.predicts) == 0 {
		g.predicts = clonePredicts(g.dict.Predicts)
	}
	seed := splicePredict(&g.predicts)
	g.sex = seed.Sex
	return strings.ToUpper(g.replaceSets(seed.Message))
}

func (g *generator) getAction() string {
	items := g.actions[g.sex]
	if len(items) == 0 {
		items = append([]string(nil), g.dict.Actions[g.sex]...)
	}
	item, rest := spliceStringSlice(items)
	g.actions[g.sex] = rest
	return strings.ToUpper(g.replaceSets(item))
}

func (g *generator) getConclusion() string {
	items := g.conclusions[g.sex]
	if len(items) == 0 {
		items = append([]string(nil), g.dict.Conclusions[g.sex]...)
	}
	item, rest := spliceStringSlice(items)
	g.conclusions[g.sex] = rest
	return strings.ToUpper(g.replaceSets(item))
}

func (g *generator) replaceSets(template string) string {
	content := template
	matches := placeholderRE.FindAllString(content, -1)
	for _, match := range matches {
		key := strings.TrimSuffix(strings.TrimPrefix(match, "["), "]")
		replacements := []string{key}
		for _, set := range g.dict.Sets {
			if set.Key == key {
				replacements = set.Replacements
				break
			}
		}
		variant := replacements[rand.Intn(len(replacements))]
		content = strings.Replace(content, match, variant, 1)
		content = strings.TrimSpace(content)
	}
	return content
}

func splicePredict(items *[]predict) predict {
	idx := rand.Intn(len(*items))
	item := (*items)[idx]
	*items = append((*items)[:idx], (*items)[idx+1:]...)
	return item
}

func spliceStringSlice(items []string) (string, []string) {
	idx := rand.Intn(len(items))
	item := items[idx]
	rest := append(items[:idx], items[idx+1:]...)
	return item, rest
}

func clonePredicts(items []predict) []predict {
	out := make([]predict, len(items))
	copy(out, items)
	return out
}

func cloneActions(items map[string][]string) map[string][]string {
	out := make(map[string][]string, len(items))
	for key, values := range items {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func cloneConclusions(items map[string][]string) map[string][]string {
	return cloneActions(items)
}
