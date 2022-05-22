package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHVSock(t *testing.T) {
	scheme, sock := parseSockStr("hvsock://00000000-0000-0000-0000-000000000000/00001010-FACB-11E6-BD58-64006A7986D3")
	assert.Equal(t, "hvsock", scheme)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000:00001010-facb-11e6-bd58-64006a7986d3", sock.String())
}
