package apple

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math/big"
)

// groupHex is the public 2048-bit SRP group from RFC 5054 Appendix A.
const groupHex = "AC6BDB41324A9A9BF166DE5E1389582FAF72B6651987EE07FC3192943DB56050A37329CBB4A099ED8193E0757767A13DD52312AB4B03310DCD7F48A9DA04FD50E8083969EDB767B0CF6095179A163AB3661A05FBD5FAAAE82918A9962F0B93B855F97993EC975EEAA80D740ADBF4FF747359D041D5C33EA71D281E446B14773BCA97B43A23FB801676BD207A436C6481F1D2B9078717461A5B9D32E688F87748544523B524B0D57D5EA77A2775D2ECFA032CFBDBF52FB3786160279004E57AE6AF874E7303CE53299CCC041C7BC308D82A5698F3A8D0C38271AE35F8E9DBFBB694B5C803D89F7AE435DE236D525F54759B65E372FCD68EF20FA7111F9E4AFF73"

// digest hashes consecutive byte strings without retaining password material.
func digest(parts ...[]byte) []byte {
	h := sha256.New()
	for _, part := range parts {
		h.Write(part)
	}
	return h.Sum(nil)
}

// srpProof computes Apple's observed SRP variant using standard SRP-6a arithmetic.
// Private exponents/derived keys are ephemeral and are never placed in session state.
func srpProof(secret []byte, account, password string, salt, server []byte, iterations int, protocol string) (string, string, error) {
	if len(secret) != 32 || len(salt) == 0 || len(salt) > 1024 || len(server) > 256 || iterations < 1 || iterations > 1000000 {
		return "", "", ErrAuth
	}
	n, _ := new(big.Int).SetString(groupHex, 16)
	g := big.NewInt(2)
	pad := func(v *big.Int) []byte { return v.FillBytes(make([]byte, 256)) }
	integer := func(b []byte) *big.Int { return new(big.Int).SetBytes(b) }
	a := integer(secret)
	b := integer(server)
	if a.Sign() == 0 || b.Sign() == 0 || b.Cmp(n) >= 0 {
		return "", "", ErrAuth
	}
	public := new(big.Int).Exp(g, a, n)
	input := digest([]byte(password))
	switch protocol {
	case "s2k":
	case "s2k_fo":
		input = []byte(hex.EncodeToString(input))
	default:
		return "", "", ErrAuth
	}
	key, err := pbkdf2.Key(sha256.New, string(input), salt, iterations, 32)
	if err != nil {
		return "", "", ErrAuth
	}
	defer clear(key)
	defer clear(input)
	x := integer(digest(salt, digest([]byte(":"), key)))
	u := integer(digest(pad(public), pad(b)))
	if u.Sign() == 0 {
		return "", "", ErrAuth
	}
	multiplier := integer(digest(pad(n), pad(g)))
	base := new(big.Int).Sub(b, new(big.Int).Mul(multiplier, new(big.Int).Exp(g, x, n)))
	exponent := new(big.Int).Add(a, new(big.Int).Mul(u, x))
	shared := new(big.Int).Exp(base, exponent, n)
	if shared.Sign() == 0 {
		return "", "", ErrAuth
	}
	sessionKey := digest(pad(shared))
	defer clear(sessionKey)
	xor := digest(pad(n))
	other := digest(pad(g))
	for i := range xor {
		xor[i] ^= other[i]
	}
	proof := digest(xor, digest([]byte(account)), salt, pad(public), pad(b), sessionKey)
	serverProof := digest(pad(public), proof, sessionKey)
	return base64.StdEncoding.EncodeToString(proof), base64.StdEncoding.EncodeToString(serverProof), nil
}

// srpStart generates one ephemeral exponent and its padded public value.
func srpStart() ([]byte, string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", ErrAuth
	}
	n, _ := new(big.Int).SetString(groupHex, 16)
	public := new(big.Int).Exp(big.NewInt(2), new(big.Int).SetBytes(secret), n)
	return secret, base64.StdEncoding.EncodeToString(public.FillBytes(make([]byte, 256))), nil
}
