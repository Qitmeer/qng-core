package x16rv3

import (
	"crypto/sha512"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/aes"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/blake"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/bmw"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/cubehash"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/echo"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/fugue"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/groestl"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/hamsi"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/hash"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/jh"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/keccak"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/luffa"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/shabal"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/shavite"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/simd"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/skein"
	"github.com/Qitmeer/qng-core/crypto/x16rv3/whirlpool"
)

const (
	BLAKE = iota
	BMW
	GROESTL
	JH
	KECCAK
	SKEIN
	LUFFA
	CUBEHASH
	SHAVITE
	SIMD
	ECHO
	HAMSI
	FUGUE
	SHABAL
	WHIRLPOOL
	SHA512
	HASH_FUNC_COUNT
)

var x16rv3_hashOrder = [HASH_FUNC_COUNT]uint{}

func aes_expand_key_soft(header []byte) [12]Uint128 {
	var keyData = make([]byte, 192)
	copy(keyData[:96], header[:96])
	var key = [12]Uint128{}
	for i := 0; i < 12; i++ {
		key[i] = FromBytes(keyData[i*16 : i*16+16])
	}
	key[6] = Xor128(key[0], key[2])
	key[7] = Xor128(key[1], key[3])
	key[8] = Xor128(key[0], key[4])
	key[9] = Xor128(key[1], key[5])
	key[10] = Xor128(key[2], key[4])
	key[11] = Xor128(key[3], key[5])
	return key
}

func get_x16rv3_order(inp []byte) []byte {
	size := 113
	input := make([]byte, size)
	if len(inp) < size {
		size = len(inp)
	}
	copy(input[:size], inp[:size])
	var ek [12]Uint128
	var endiandata [128]byte
	copy(endiandata[:113], input[:113])
	ek = aes_expand_key_soft(input[4:])
	var aesdata = [12]Uint128{}
	var data_in [8]Uint128
	for i := 0; i < 8; i++ {
		data_in[i] = FromBytes(endiandata[i*16 : i*16+16])
	}
	for j := 0; j < 8; j++ {
		aesdata[j] = FromIntsArray(aes.Aes_enc_soft(aesdata[j].ToUint64(), data_in[j].ToUint64(), ek[j].ToUint64()))
	}
	var xor_out = Ur128_5xor(aesdata[4], aesdata[5], aesdata[6], aesdata[7], aesdata[0])
	aesdata[8] = FromIntsArray(aes.Aes_enc_soft(aesdata[8].ToUint64(), xor_out.ToUint64(), ek[8].ToUint64()))
	xor_out = Ur128_5xor(aesdata[4], aesdata[5], aesdata[6], aesdata[7], aesdata[1])
	aesdata[9] = FromIntsArray(aes.Aes_enc_soft(aesdata[9].ToUint64(), xor_out.ToUint64(), ek[9].ToUint64()))
	xor_out = Ur128_5xor(aesdata[4], aesdata[5], aesdata[6], aesdata[7], aesdata[2])
	aesdata[10] = FromIntsArray(aes.Aes_enc_soft(aesdata[10].ToUint64(), xor_out.ToUint64(), ek[10].ToUint64()))
	xor_out = Ur128_5xor(aesdata[4], aesdata[5], aesdata[6], aesdata[7], aesdata[3])
	aesdata[11] = FromIntsArray(aes.Aes_enc_soft(aesdata[11].ToUint64(), xor_out.ToUint64(), ek[11].ToUint64()))
	outPut := ArrayToBytes(aesdata[8:])
	aesData6 := aesdata[6].GetBytes()
	for k := 0; k < 16; k++ {
		x16rv3_hashOrder[k] = uint(aesData6[k] & 0x0f)
	}
	return outPut
}

// Hash contains the state objects
// required to perform the x16.Hash.
type Hash struct {
	tha [64]byte
	thb [64]byte

	blake    hash.Digest
	bmw      hash.Digest
	cubehash hash.Digest
	echo     hash.Digest
	groestl  hash.Digest
	jh       hash.Digest
	keccak   hash.Digest
	luffa    hash.Digest
	shavite  hash.Digest
	simd     hash.Digest
	skein    hash.Digest
}

// New returns a new object to compute a x16 hash.
func New() *Hash {
	ref := &Hash{}

	ref.blake = blake.New()
	ref.bmw = bmw.New()
	ref.cubehash = cubehash.New()
	ref.echo = echo.New()
	ref.groestl = groestl.New()
	ref.jh = jh.New()
	ref.keccak = keccak.New()
	ref.luffa = luffa.New()
	ref.shavite = shavite.New()
	ref.simd = simd.New()
	ref.skein = skein.New()

	return ref
}

// Hash computes the hash from the src bytes and stores the result in dst.
func (ref *Hash) Hash(src []byte, dst []byte) {
	outHash := get_x16rv3_order(src)
	in := ref.tha[:]
	copy(in[:], outHash[:])
	out := ref.thb[:]
	for i := 0; i < 16; i++ {
		switch x16rv3_hashOrder[i] {
		case BLAKE:
			ref.blake.Write(in)
			ref.blake.Close(out, 0, 0)
			copy(in, out)
		case BMW:
			ref.bmw.Write(in)
			ref.bmw.Close(out, 0, 0)
			copy(in, out)
		case GROESTL:
			ref.groestl.Write(in)
			ref.groestl.Close(out, 0, 0)
			copy(in, out)
		case SKEIN:
			ref.skein.Write(in)
			ref.skein.Close(out, 0, 0)
			copy(in, out)
		case JH:
			ref.jh.Write(in)
			ref.jh.Close(out, 0, 0)
			copy(in, out)
		case KECCAK:
			ref.keccak.Write(in)
			ref.keccak.Close(out, 0, 0)
			copy(in, out)
		case LUFFA:
			ref.luffa.Write(in)
			ref.luffa.Close(out, 0, 0)
			copy(in, out)
		case CUBEHASH:
			ref.cubehash.Write(in)
			ref.cubehash.Close(out, 0, 0)
			copy(in, out)
		case SHAVITE:
			ref.shavite.Write(in)
			ref.shavite.Close(out, 0, 0)
			copy(in, out)
		case SIMD:
			ref.simd.Write(in)
			ref.simd.Close(out, 0, 0)
			copy(in, out)
		case ECHO:
			ref.echo.Write(in)
			ref.echo.Close(out, 0, 0)
			copy(in, out)
		case HAMSI:
			hamsi.Sph_hamsi512_process(in[:], out[:], 64)
			copy(in, out)
		case FUGUE:
			fugue.Sph_fugue512_process(in[:], out[:], 64)
			copy(in, out)
		case SHABAL:
			shabal.Shabal_512_process(in[:], out[:], 64)
			copy(in, out)
		case WHIRLPOOL:
			wh := whirlpool.New()
			wh.Write(in)
			out = wh.Sum(nil)
			copy(in, out)
		case SHA512:
			sha := sha512.New()
			sha.Write(in)
			out = sha.Sum(nil)
			copy(in, out)
		}
	}
	copy(dst, in[:len(dst)])
}

func Sum256(in []byte) [32]byte {
	out := make([]byte, 32)
	x16rv3 := New()
	x16rv3.Hash(in, out)
	var sum [32]byte
	copy(sum[:], out[:32])
	return sum
}

func Sum512(in []byte) [64]byte {
	out := make([]byte, 64)
	x16rv3 := New()
	x16rv3.Hash(in, out)
	var sum [64]byte
	copy(sum[:], out[:])
	return sum
}
