package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WordWrap(t *testing.T) {
	s := "NEWS PLAY ELSE CABLE UNLOCK SUSPECT TOAST MIXTURE SCARE POTTERY MONTH ESSAY IMMUNE BURGER STING RATE PLANET TOWN"

	assert.Equal(t, "NEWS PLAY ELSE CABLE UNLOCK SUSPECT\nTOAST MIXTURE SCARE POTTERY MONTH ESSAY\nIMMUNE BURGER STING RATE PLANET TOWN", WordWrap(s, 40))
	assert.Equal(t, "NEWS PLAY ELSE CABLE UNLOCK SUSPECT TOAST MIXTURE SCARE\nPOTTERY MONTH ESSAY IMMUNE BURGER STING RATE PLANET TOWN", WordWrap(s, 60))
	assert.Equal(t, "NEWS PLAY\nELSE CABLE\nUNLOCK\nSUSPECT\nTOAST\nMIXTURE\nSCARE\nPOTTERY\nMONTH\nESSAY\nIMMUNE\nBURGER\nSTING RATE\nPLANET\nTOWN", WordWrap(s, 10))
	assert.Equal(t, "\nNEWS\nPLAY\nELSE\nCABLE\nUNLOCK\nSUSPECT\nTOAST\nMIXTURE\nSCARE\nPOTTERY\nMONTH\nESSAY\nIMMUNE\nBURGER\nSTING\nRATE\nPLANET\nTOWN", WordWrap(s, 3))
}
