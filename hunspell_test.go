// Copyright ©2022 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hunspell

import (
	"bufio"
	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

const (
	// Language used for testing.
	lang = "en_US"

	// Path to dictionaries.
	path = "/usr/share/hunspell"
)

var (
	words = []string{
		"children",
		"language",
		"necessarily",
		"necessary",
		"ran",
		"unnecessarily",
		"unnecessary",
		"unwritten",

		"definately",
		"langauge",
		"seperate",
	}
	wantSuggest = map[string][]string{
		"children": {
			"children",
		},
		"language": {
			"language", "languages",
		},
		"necessarily": {
			"necessarily",
		},
		"necessary": {
			"necessary",
		},
		"ran": {
			"rein", "tan", "ram", "an", "rain", "rant", "roan", "rand",
			"rang", "gran", "bran", "rank", "Iran", "Tran", "Oran",
		},
		"unnecessarily": {
			"unnecessarily",
		},
		"unnecessary": {
			"unnecessary",
		},
		"unwritten": {
			"unwritten",
		},

		"definately": {
			"definitely", "effeminately", "definitively", "indefinably",
		},
		"langauge": {
			"language", "Augean", "Angela",
		},
		"seperate": {
			"separate", "desperate", "temperate", "exasperate", "serrate",
		},
	}
)

func TestHunspell(t *testing.T) {
	if _, err := exec.LookPath("hunspell"); err != nil {
		t.Fatalf("hunspell not available for testing: %v", err)
	}

	s, err := NewSpell(path, lang)
	if err != nil {
		t.Fatalf("failed to open dictionary: %v", err)
	}

	t.Run("analyze", func(t *testing.T) {
		want, err := hunspell("analyze", words)
		if err != nil {
			t.Errorf("unexpected error getting want values: %v", err)
			return
		}
		for _, w := range words {
			got := s.Analyze(w)
			if !reflect.DeepEqual(got, want[w]) {
				t.Errorf("unexpected result for analyze %q: got:%#v want:%#v",
					w, got, want[w])
			}
		}
	})

	t.Run("spell", func(t *testing.T) {
		want, err := hunspell("spell", words)
		if err != nil {
			t.Errorf("unexpected error getting want values: %v", err)
			return
		}
		for _, w := range words {
			gotCorrect := s.IsCorrect(w)
			wantCorrect := want[w][0] == "*" || want[w][0] == "+"
			if gotCorrect != wantCorrect {
				t.Errorf("unexpected result for %q is correct: got:%t want:%t",
					w, gotCorrect, wantCorrect)
			}
		}
	})

	t.Run("stem", func(t *testing.T) {
		want, err := hunspell("stem", words)
		if err != nil {
			t.Errorf("unexpected error getting want values: %v", err)
			return
		}
		for _, w := range words {
			got := s.Stem(w)
			if !reflect.DeepEqual(got, want[w]) {
				t.Errorf("unexpected result for stem %q: got:%s want:%s",
					w, got, want[w])
			}
		}
	})

	t.Run("suggest", func(t *testing.T) {
		for _, w := range words {
			got := s.Suggest(w)
			if !reflect.DeepEqual(got, wantSuggest[w]) {
				t.Errorf("unexpected result for stem %q: got:%s want:%s",
					w, got, wantSuggest[w])
			}
		}
	})

	t.Run("add_remove", func(t *testing.T) {
		s.Add("seperate")
		if !s.IsCorrect("seperate") {
			t.Error("added word still incorrect")
		}
		s.Remove("seperate")
		if s.IsCorrect("seperate") {
			t.Error("removed word still correct")
		}
		s.AddWithAffix("seperate", "separate")
		if !s.IsCorrect("seperate") {
			t.Error("added word with affix still incorrect")
		}
		s.Remove("seperate")
		if s.IsCorrect("seperate") {
			t.Error("removed word still correct")
		}
	})

	t.Run("add_dict", func(t *testing.T) {
		if s.IsCorrect("colour") {
			t.Error("absent word is incorrectly accepted")
		}
		err := s.AddDict("en_au.dic")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if !s.IsCorrect("colour") {
			t.Error("word added by dictionary is still incorrect")
		}
	})
}

func hunspell(action string, words []string) (map[string][]string, error) {
	flags := map[string]string{
		"analyze": "-m",
		"spell":   "",
		"stem":    "-s",
	}

	cmd := exec.Command("hunspell")
	cmd.Stdin = strings.NewReader("")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(buf.String())

	cmd = exec.Command("hunspell", "-d", lang, flags[action])
	cmd.Stdin = strings.NewReader(strings.Join(words, " "))
	buf.Reset()
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	results := make(map[string][]string)
	sc := bufio.NewScanner(&buf)
	for i := 0; sc.Scan(); {
		text := sc.Text()
		if text == "" || text == version {
			continue
		}
		switch action {
		case "analyze":
			r := strings.SplitN(text, " ", 2)
			if len(r) > 1 {
				results[r[0]] = r[1:]
			}
		case "spell":
			results[words[i]] = strings.Split(text, " ")
		case "stem":
			r := strings.Split(text, " ")
			if len(r) > 1 {
				results[r[0]] = r[1:]
			}
		}
		i++
	}
	return results, sc.Err()
}
