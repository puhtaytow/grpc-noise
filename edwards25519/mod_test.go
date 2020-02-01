// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edwards25519

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

type zeroReader struct{}

func (zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func TestUnmarshalMarshal(t *testing.T) {
	pub, _, _ := GenerateKey(rand.Reader)

	var A ExtendedGroupElement

	var publicKey1 PublicKey
	copy(publicKey1[:], pub[:])

	if !A.FromBytes((*[SizePublicKey]byte)(&publicKey1)) {
		t.Fatalf("ExtendedGroupElement.FromBytes failed")
	}

	var publicKey2 PublicKey
	A.ToBytes((*[SizePublicKey]byte)(&publicKey2))

	if publicKey1 != publicKey2 {
		t.Errorf("FromBytes(%v)->ToBytes does not round-trip, got %x\n", publicKey1, publicKey2)
	}
}

func TestSignVerify(t *testing.T) {
	var zero zeroReader
	public, private, _ := GenerateKey(zero)

	message := []byte("test message")
	sig := Sign(private, message)
	if !Verify(public, message, sig) {
		t.Errorf("valid signature rejected")
	}

	wrongMessage := []byte("wrong message")
	if Verify(public, wrongMessage, sig) {
		t.Errorf("signature of different message accepted")
	}
}

func TestCryptoSigner(t *testing.T) {
	var zero zeroReader

	public, private, _ := GenerateKey(zero)
	public2 := private.Public()

	if public != public2 {
		t.Errorf("public keys do not match: original:%x vs Public():%x", public, public2)
	}

	message := []byte("message")
	signature, err := private.Sign(message, crypto.Hash(0))

	if err != nil {
		t.Fatalf("error from Sign(): %s", err)
	}

	if !Verify(public, message, signature) {
		t.Errorf("Verify failed on signature from Sign()")
	}
}

func TestGolden(t *testing.T) {
	// sign.input.gz is a selection of test cases from
	// https://ed25519.cr.yp.to/python/sign.input
	testDataZ, err := os.Open("testdata/sign.input.gz")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = testDataZ.Close()
	}()

	testData, err := gzip.NewReader(testDataZ)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = testData.Close()
	}()

	scanner := bufio.NewScanner(testData)
	lineNo := 0

	for scanner.Scan() {
		lineNo++

		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 5 {
			t.Fatalf("bad number of parts on line %d", lineNo)
		}

		privBytes, _ := hex.DecodeString(parts[0])
		pubBytes, _ := hex.DecodeString(parts[1])
		msg, _ := hex.DecodeString(parts[2])
		sig, _ := hex.DecodeString(parts[3])
		// The signatures in the test vectors also include the message
		// at the end, but we just want R and S.
		sig = sig[:SizeSignature]

		if l := len(pubBytes); l != SizePublicKey {
			t.Fatalf("bad public key length on line %d: got %d bytes", lineNo, l)
		}

		var priv PrivateKey
		copy(priv[:], privBytes)
		copy(priv[32:], pubBytes)

		sig2 := Sign(priv, msg)
		if !bytes.Equal(sig, sig2[:]) {
			t.Errorf("different signature result on line %d: %x vs %x", lineNo, sig, sig2)
		}

		var pub PublicKey
		copy(pub[:], pubBytes)

		if !Verify(pub, msg, sig2) {
			t.Errorf("signature failed to verify on line %d", lineNo)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading test data: %s", err)
	}
}

func BenchmarkKeyGeneration(b *testing.B) {
	var zero zeroReader
	for i := 0; i < b.N; i++ {
		if _, _, err := GenerateKey(zero); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigning(b *testing.B) {
	var zero zeroReader

	_, priv, err := GenerateKey(zero)
	if err != nil {
		b.Fatal(err)
	}
	message := []byte("Hello, world!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(priv, message)
	}
}

func BenchmarkVerification(b *testing.B) {
	var zero zeroReader

	pub, priv, err := GenerateKey(zero)
	if err != nil {
		b.Fatal(err)
	}
	message := []byte("Hello, world!")
	signature := Sign(priv, message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(pub, message, signature)
	}
}
