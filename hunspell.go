// Copyright ©2022 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hunspell provides bindings to the hunspell spelling library.
package hunspell

// #cgo pkg-config: hunspell
// #include <stdlib.h>
// #include <hunspell/hunspell.h>
import "C"

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

// Spell is a hunspell spelling checker. A Spell is only valid if it is
// returned by a successful call to NewSpell.
type Spell struct {
	handle *C.Hunhandle
}

// NewSpell returns a spelling checker initialized with the dictionary
// specified by the lang key located in the given path. NewSpell called
// with both parameters empty will return a valid empty spell checker.
// If lang is empty and path is not, an error is returned. NewSpell checks
// for the existence of the dictionary files.
func NewSpell(path, lang string) (*Spell, error) {
	if lang == "" && path != "" {
		return nil, errors.New("missing lang")
	}
	var affPath, dictPath string
	if lang != "" {
		affPath = filepath.Join(path, lang+".aff")
		_, err := os.Stat(affPath)
		if err != nil {
			pe := err.(*os.PathError)
			pe.Op = "open"
			return nil, err
		}
		dictPath = filepath.Join(path, lang+".dic")
		_, err = os.Stat(dictPath)
		if err != nil {
			pe := err.(*os.PathError)
			pe.Op = "open"
			return nil, err
		}
	}
	aff := C.CString(affPath)
	dict := C.CString(dictPath)
	s := &Spell{handle: C.Hunspell_create(aff, dict)}
	runtime.SetFinalizer(s, func(h *Spell) {
		C.Hunspell_destroy(h.handle)
	})
	C.free(unsafe.Pointer(aff))
	C.free(unsafe.Pointer(dict))
	return s, nil
}

// IsCorrect returns whether the provided word is spelled correctly.
func (s *Spell) IsCorrect(word string) bool {
	w := C.CString(word)
	defer C.free(unsafe.Pointer(w))
	return C.Hunspell_spell(s.handle, w) != 0
}

// Suggest returns suggestions for the provided word.
func (s *Spell) Suggest(word string) []string {
	w := C.CString(word)
	var words **C.char
	n := C.Hunspell_suggest(s.handle, &words, w)
	C.free(unsafe.Pointer(w))
	defer C.Hunspell_free_list(s.handle, &words, n)
	return goStrings(words, n)
}

// Add adds the provided word to the run-time dictionary.
func (s *Spell) Add(word string) (ok bool) {
	w := C.CString(word)
	defer C.free(unsafe.Pointer(w))
	return C.Hunspell_add(s.handle, w) == 0
}

// AddWithAffix adds the provided word to the run-time dictionary including
// affix information from the dictionary example word.
func (s *Spell) AddWithAffix(word, example string) (ok bool) {
	w := C.CString(word)
	defer C.free(unsafe.Pointer(w))
	e := C.CString(example)
	defer C.free(unsafe.Pointer(e))
	return C.Hunspell_add_with_affix(s.handle, w, e) == 0
}

// Remove removes the provided word from the run-time dictionary.
func (s *Spell) Remove(word string) (ok bool) {
	w := C.CString(word)
	defer C.free(unsafe.Pointer(w))
	return C.Hunspell_remove(s.handle, w) == 0
}

// AddDict adds extra dictionary (.dic file) to the run-time dictionary.
func (s *Spell) AddDict(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		pe := err.(*os.PathError)
		pe.Op = "open"
		return err
	}
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	if C.Hunspell_add_dic(s.handle, p) == 1 {
		return errors.New("failed to add dictionary")
	}
	return nil
}

// Analyze returns a morphological analysis of the word.
func (s *Spell) Analyze(word string) []string {
	w := C.CString(word)
	var words **C.char
	n := C.Hunspell_analyze(s.handle, &words, w)
	C.free(unsafe.Pointer(w))
	defer C.Hunspell_free_list(s.handle, &words, n)
	return goStrings(words, n)
}

// Stem returns the stems of the provided word.
func (s *Spell) Stem(word string) []string {
	w := C.CString(word)
	var words **C.char
	n := C.Hunspell_stem(s.handle, &words, w)
	C.free(unsafe.Pointer(w))
	defer C.Hunspell_free_list(s.handle, &words, n)
	return goStrings(words, n)
}

func goStrings(src **C.char, n C.int) []string {
	if n == 0 {
		return nil
	}
	dst := make([]string, 0, n)
	for _, v := range unsafe.Slice(src, n) {
		dst = append(dst, C.GoString(v))
	}
	return dst
}

/*
generate appears to be broken and I have also not been able to get generate2
to work.

See https://github.com/hunspell/hunspell/issues/554

func (s *Spell) ByExample(word, example string) []string {
	w := C.CString(word)
	e := C.CString(example)
	var words **C.char
	n := C.Hunspell_generate(s.handle, &words, w, e)
	C.free(unsafe.Pointer(w))
	C.free(unsafe.Pointer(e))
	defer C.Hunspell_free_list(s.handle, &words, n)
	return goStrings(words, n)
}

func (s *Spell) ByDescription(word string, descriptions []string, n int) []string {
	w := C.CString(word)
	d, cFreeD := cStrings(descriptions)
	var words **C.char
	_n := C.Hunspell_generate2(s.handle, &words, w, d, C.int(n))
	C.free(unsafe.Pointer(w))
	cFreeD()
	defer C.Hunspell_free_list(s.handle, &words, _n)
	return goStrings(words, _n)
}

func cStrings(src []string) (p **C.char, free func()) {
	if len(src) == 0 {
		return nil, func() {}
	}
	dst := make([]*C.char, len(src))
	for i, s := range src {
		dst[i] = C.CString(s)
	}
	free = func() {
		for _, p := range dst {
			C.free(unsafe.Pointer(p))
		}
	}
	return &dst[0], free
}
*/
