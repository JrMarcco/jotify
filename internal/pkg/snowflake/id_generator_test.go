package snowflake

import (
	"strconv"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/stretchr/testify/assert"
)

func TestGenerator_NextId(t *testing.T) {
	t.Parallel()

	g := NewGenerator()

	bizId := uint64(2025)
	bizKey := "test_biz_key:" + strconv.FormatUint(bizId, 10)

	id := g.NextId(bizId, bizKey)

	hashVal := ExtractHash(id)
	if hashVal >= 1024 {
		t.Errorf("hashVal should be in range [0, 1024), but got %d", hashVal)
	}

	wantHashVal := xxhash.Sum64String(HashKey(bizId, bizKey))
	assert.Equal(t, hashVal, wantHashVal%1024)

	seq := ExtractSequence(id)
	if seq >= (1 << 12) {
		t.Errorf("seq should be in range [0, 4096), but got %d", seq)
	}
	assert.Equal(t, seq, uint64(0))

}

func TestGenerator_NextId_Uniqueness(t *testing.T) {
	t.Parallel()
	g := NewGenerator()

	idCnt := 1000000
	ids := make(map[uint64]struct{}, idCnt)

	for i := 0; i < idCnt; i++ {
		bizId := uint64(i % 100)                           // reuse bizId to test uniqueness
		bizKey := "test_biz_key:" + string(rune('A'+i%26)) // reuse bizKey to test uniqueness

		id := g.NextId(bizId, bizKey)

		if _, exists := ids[id]; exists {
			t.Logf("id %d already exists", id)
		}
		ids[id] = struct{}{}
	}

	conflictCnt := idCnt - len(ids)

	t.Logf("generated %d ids", idCnt)
	t.Logf("%d id conflicts and %.2f%% of the time. ",
		conflictCnt,
		float64(conflictCnt)/float64(idCnt)*float64(100),
	)
}

func TestGenerator_NextId_SeqIncr(t *testing.T) {
	t.Parallel()
	g := NewGenerator()

	biz := uint64(2025)
	bizKey := "test_biz_key:" + strconv.FormatUint(biz, 10)

	cnt := 100
	ids := make([]uint64, 0, cnt)
	for i := 0; i < cnt; i++ {
		id := g.NextId(biz, bizKey)

		seq := ExtractSequence(id)
		assert.Equal(t, seq, uint64(i))

		ids = append(ids, id)
	}

	wantHashVal := ExtractHash(ids[0])
	for i := 1; i < cnt; i++ {
		hashVal := ExtractHash(ids[i])
		assert.Equal(t, hashVal, wantHashVal)
	}
}

func TestGenerator_NextId_SeqRollover(t *testing.T) {
	t.Parallel()
	g := NewGenerator()

	// change seq to max value
	g.sequence = sequenceMask

	bizId := uint64(2025)
	bizKey := "test_biz_key:" + strconv.FormatUint(bizId, 10)

	id := g.NextId(bizId, bizKey)

	seq := ExtractSequence(id)
	assert.Equal(t, seq, sequenceMask, "sequence should be max value")

	id = g.NextId(bizId, bizKey)
	seq = ExtractSequence(id)
	assert.Equal(t, seq, uint64(0), "sequence should be rolled over to 0")
}
