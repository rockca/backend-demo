package transaction

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

var (
	ErrInvalidLength = errors.New("invalid signature length")
)

type Signer interface {
	// Sign signs data with ethereum prefix (eip191 type 0x45).
	Sign(data []byte) ([]byte, error)
	// SignTx signs an ethereum transaction.
	SignTx(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error)
	// PublicKey returns the public key this signer uses.
	PublicKey() (*ecdsa.PublicKey, error)
	// EthereumAddress returns the ethereum address this signer uses.
	EthereumAddress() (common.Address, error)
}

func LegacyKeccak256(data []byte) ([]byte, error) {
	var err error
	hasher := sha3.NewLegacyKeccak256()
	_, err = hasher.Write(data)
	if err != nil {
		return nil, err
	}
	return hasher.Sum(nil), err
}

// addEthereumPrefix adds the ethereum prefix to the data.
func addEthereumPrefix(data []byte) []byte {
	return []byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data))
}

// hashWithEthereumPrefix returns the hash that should be signed for the given data.
func hashWithEthereumPrefix(data []byte) ([]byte, error) {
	return LegacyKeccak256(addEthereumPrefix(data))
}

// Recover verifies signature with the data base provided.
// It is using `btcec.RecoverCompact` function.
func Recover(signature, data []byte) (*ecdsa.PublicKey, error) {
	if len(signature) != 65 {
		return nil, ErrInvalidLength
	}
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = signature[64]
	copy(btcsig[1:], signature)

	hash, err := hashWithEthereumPrefix(data)
	if err != nil {
		return nil, err
	}

	p, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, hash)
	return (*ecdsa.PublicKey)(p), err
}

type defaultSigner struct {
	key *ecdsa.PrivateKey
}

func NewDefaultSigner(key *ecdsa.PrivateKey) Signer {
	return &defaultSigner{
		key: key,
	}
}

// PublicKey returns the public key this signer uses.
func (d *defaultSigner) PublicKey() (*ecdsa.PublicKey, error) {
	return &d.key.PublicKey, nil
}

// Sign signs data with ethereum prefix (eip191 type 0x45).
func (d *defaultSigner) Sign(data []byte) (signature []byte, err error) {
	hash, err := hashWithEthereumPrefix(data)
	if err != nil {
		return nil, err
	}

	return d.sign(hash, true)
}

// SignTx signs an ethereum transaction.
func (d *defaultSigner) SignTx(transaction *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	txSigner := types.NewEIP155Signer(chainID)
	hash := txSigner.Hash(transaction).Bytes()
	// isCompressedKey is false here so we get the expected v value (27 or 28)
	signature, err := d.sign(hash, false)
	if err != nil {
		return nil, err
	}

	// v value needs to be adjusted by 27 as transaction.WithSignature expects it to be 0 or 1
	signature[64] -= 27
	return transaction.WithSignature(txSigner, signature)
}

// EthereumAddress returns the ethereum address this signer uses.
func (d *defaultSigner) EthereumAddress() (common.Address, error) {
	publicKey, err := d.PublicKey()
	if err != nil {
		return common.Address{}, err
	}
	eth, err := NewEthereumAddress(*publicKey)
	if err != nil {
		return common.Address{}, err
	}
	var ethAddress common.Address
	copy(ethAddress[:], eth)
	return ethAddress, nil
}

// sign the provided hash and convert it to the ethereum (r,s,v) format.
func (d *defaultSigner) sign(sighash []byte, isCompressedKey bool) ([]byte, error) {
	signature, err := btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(d.key), sighash, false)
	if err != nil {
		return nil, err
	}

	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := signature[0]
	copy(signature, signature[1:])
	signature[64] = v
	return signature, nil
}

// NewEthereumAddress returns a binary representation of ethereum blockchain address.
// This function is based on github.com/ethereum/go-ethereum/crypto.PubkeyToAddress.
func NewEthereumAddress(p ecdsa.PublicKey) ([]byte, error) {
	if p.X == nil || p.Y == nil {
		return nil, errors.New("invalid public key")
	}
	pubBytes := elliptic.Marshal(btcec.S256(), p.X, p.Y)
	pubHash, err := LegacyKeccak256(pubBytes[1:])
	if err != nil {
		return nil, err
	}
	return pubHash[12:], err
}
