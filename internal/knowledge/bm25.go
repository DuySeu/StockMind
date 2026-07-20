package knowledge

import (
	"hash/crc32"
	"math"
	"strings"
	"sync"
	"unicode"
)

// Tokenizer generates BM25 sparse vectors using hash-based term indexing.
type Tokenizer struct {
	vocabSize uint32
	mu        sync.RWMutex
	docCount  int
	df        map[uint32]int // document frequency per term hash
}

// NewTokenizer creates a BM25 tokenizer with the given vocabulary size.
func NewTokenizer(vocabSize uint32) *Tokenizer {
	if vocabSize == 0 {
		vocabSize = 30000
	}
	return &Tokenizer{
		vocabSize: vocabSize,
		df:        make(map[uint32]int),
	}
}

// AddDocument updates document frequency counts for IDF calculation.
func (t *Tokenizer) AddDocument(text string) {
	terms := tokenize(text)
	seen := make(map[uint32]bool)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.docCount++
	for _, term := range terms {
		idx := t.hash(term)
		if !seen[idx] {
			seen[idx] = true
			t.df[idx]++
		}
	}
}

// AddDocuments updates document frequency counts for multiple documents.
func (t *Tokenizer) AddDocuments(texts []string) {
	for _, text := range texts {
		t.AddDocument(text)
	}
}

// Vectorize converts a text chunk into a sparse vector using TF-IDF weights.
func (t *Tokenizer) Vectorize(text string) SparseVector {
	terms := tokenize(text)
	if len(terms) == 0 {
		return SparseVector{}
	}

	// Compute term frequency
	tf := make(map[uint32]float32)
	for _, term := range terms {
		tf[t.hash(term)]++
	}

	t.mu.RLock()
	docCount := t.docCount
	df := t.df
	t.mu.RUnlock()

	if docCount == 0 {
		docCount = 1
	}

	indices := make([]uint32, 0, len(tf))
	values := make([]float32, 0, len(tf))

	for idx, freq := range tf {
		// BM25-style TF: tf / (tf + 1)
		normalizedTF := freq / (freq + 1.0)

		// IDF: log(N / (df + 1))
		docFreq := df[idx]
		if docFreq == 0 {
			docFreq = 1
		}
		idf := float32(math.Log(float64(docCount) / float64(docFreq+1)))
		if idf < 0 {
			idf = 0
		}

		score := normalizedTF * idf
		if score > 0 {
			indices = append(indices, idx)
			values = append(values, score)
		}
	}

	return SparseVector{Indices: indices, Values: values}
}

// VectorizeQuery converts a query into a sparse vector (uses same logic as Vectorize).
func (t *Tokenizer) VectorizeQuery(query string) SparseVector {
	return t.Vectorize(query)
}

// VectorizeBatch converts multiple chunks into sparse vectors.
func (t *Tokenizer) VectorizeBatch(texts []string) []SparseVector {
	results := make([]SparseVector, len(texts))
	for i, text := range texts {
		results[i] = t.Vectorize(text)
	}
	return results
}

// DocCount returns the number of documents added.
func (t *Tokenizer) DocCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.docCount
}

func (t *Tokenizer) hash(term string) uint32 {
	return crc32.ChecksumIEEE([]byte(term)) % t.vocabSize
}

// tokenize splits text into lowercase terms, removing punctuation and stopwords.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	terms := make([]string, 0, len(words))
	for _, w := range words {
		if len(w) < 2 || stopwords[w] {
			continue
		}
		terms = append(terms, w)
	}
	return terms
}

// stopwords for Vietnamese and English.
var stopwords = map[string]bool{
	// English
	"the": true, "is": true, "at": true, "which": true, "on": true,
	"a": true, "an": true, "and": true, "or": true, "but": true,
	"in": true, "of": true, "to": true, "for": true, "with": true,
	"it": true, "this": true, "that": true, "are": true, "was": true,
	"be": true, "has": true, "have": true, "had": true, "not": true,
	"from": true, "by": true, "as": true, "do": true, "if": true,
	// Vietnamese
	"là": true, "và": true, "của": true, "có": true, "được": true,
	"cho": true, "với": true, "các": true, "này": true, "trong": true,
	"từ": true, "đã": true, "không": true, "những": true, "một": true,
	"để": true, "theo": true, "về": true, "khi": true, "đến": true,
	"cũng": true, "như": true, "tại": true, "hay": true, "còn": true,
	"thì": true, "mà": true, "nên": true, "vì": true,
}
