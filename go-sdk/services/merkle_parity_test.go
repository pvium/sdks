package services

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestMerkleTreeMatchesMerkleTreeJSSortPairs(t *testing.T) {
	t.Parallel()

	leaves := [][]byte{
		mustHexBytes(t, "0x3ac225168df54212a25c1c01fd35bebfea408fdac2e31ddd6f80a4bbf9a5f1cb"),
		mustHexBytes(t, "0xb5553de315e0edf504d9150af82dafa5c4667fa618ed0a6f19c69b41166c5510"),
		mustHexBytes(t, "0x0b42b6393c1f53060fe3ddbfcd7aadcca894465a5a438f69c87d790b2299b9b2"),
	}
	levels := buildMerkleLevels(leaves)
	root := "0x" + hex.EncodeToString(levels[len(levels)-1][0])
	if root != "0x5842148bc6ebeb52af882a317c765fccd3ae80589b21a9b8cbf21abb630e46a7" {
		t.Fatalf("root mismatch: %s", root)
	}

	expectedProofs := [][]string{
		{
			"0xb5553de315e0edf504d9150af82dafa5c4667fa618ed0a6f19c69b41166c5510",
			"0x0b42b6393c1f53060fe3ddbfcd7aadcca894465a5a438f69c87d790b2299b9b2",
		},
		{
			"0x3ac225168df54212a25c1c01fd35bebfea408fdac2e31ddd6f80a4bbf9a5f1cb",
			"0x0b42b6393c1f53060fe3ddbfcd7aadcca894465a5a438f69c87d790b2299b9b2",
		},
		{
			"0x805b21d846b189efaeb0377d6bb0d201b3872a363e607c25088f025b0c6ae1f8",
		},
	}
	for i, expected := range expectedProofs {
		if got := merkleProof(levels, i); !reflect.DeepEqual(got, expected) {
			t.Fatalf("proof %d mismatch: got %+v want %+v", i, got, expected)
		}
	}
}

func mustHexBytes(t *testing.T, value string) []byte {
	t.Helper()
	out, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		t.Fatal(err)
	}
	return out
}
