package objects

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"

	"github.com/ishbir/bmgo/bitmessage/protocol/types"
)

// When a node has the hash of a public key (from a version <= 3 address) but
// not the public key itself, it must send out a request for the public key.
type GetpubkeyV3 struct {
	// The ripemd hash of the public key. This field is only included when the
	// address version is <= 3.
	Ripe [20]byte
}

func (obj *GetpubkeyV3) Serialize() []byte {
	return obj.Ripe[:]
}

func (obj *GetpubkeyV3) DeserializeReader(b io.Reader) error {
	temp, err := ioutil.ReadAll(b)
	if err != nil || len(temp) != 20 {
		return types.DeserializeFailedError("ripe")
	}
	copy(obj.Ripe[:], temp)
	return nil
}

// When a node has the hash of a public key (from a version >= 4 address) but
// not the public key itself, it must send out a request for the public key.
type GetpubkeyV4 struct {
	// The tag derived from the address version, stream number, and ripe. This
	// field is only included when the address version is >= 4.
	Tag [32]byte
}

func (obj *GetpubkeyV4) Serialize() []byte {
	return obj.Tag[:]
}

func (obj *GetpubkeyV4) DeserializeReader(b io.Reader) error {
	temp, err := ioutil.ReadAll(b)
	if err != nil || len(temp) != 32 {
		return types.DeserializeFailedError("tag")
	}
	copy(obj.Tag[:], temp)
	return nil
}

// Version 2, 3 and 4 public keys
type Pubkey interface {
	// What version of the pubkey is it?
	Version() int
	// Is the pubkey encrypted?
	IsEncrypted() bool
}

// A version 2 pubkey. This is still in use and supported by current clients but
// new v2 addresses are not generated by clients.
type PubkeyV2 struct {
	// A bitfield of optional behaviors and features that can be expected from
	// the node receiving the message.
	Behaviour uint32
	// The ECC public key used for signing (uncompressed format; normally
	// prepended with \x04 )
	PubSigningKey [64]byte
	// The ECC public key used for encryption (uncompressed format; normally
	// prepended with \x04 )
	PubEncryptionKey [64]byte
}

func (obj *PubkeyV2) Serialize() []byte {
	var b bytes.Buffer

	binary.Write(&b, binary.BigEndian, obj.Behaviour)
	b.Write(obj.PubSigningKey[:])
	b.Write(obj.PubEncryptionKey[:])

	return b.Bytes()
}

func (obj *PubkeyV2) DeserializeReader(b io.Reader) error {
	err := binary.Read(b, binary.BigEndian, &obj.Behaviour)
	if err != nil {
		return types.DeserializeFailedError("behaviour")
	}
	err = binary.Read(b, binary.BigEndian, obj.PubSigningKey[:])
	if err != nil {
		return types.DeserializeFailedError("pubSigningKey")
	}
	err = binary.Read(b, binary.BigEndian, obj.PubEncryptionKey[:])
	if err != nil {
		return types.DeserializeFailedError("pubEncryptionKey")
	}

	return nil
}

// A version 3 pubkey
type PubkeyV3 struct {
	// A bitfield of optional behaviors and features that can be expected from
	// the node receiving the message.
	Behaviour uint32
	// The ECC public key used for signing (uncompressed format; normally
	// prepended with \x04 )
	PubSigningKey [64]byte
	// The ECC public key used for encryption (uncompressed format; normally
	// prepended with \x04 )
	PubEncryptionKey [64]byte
	// Used to calculate the difficulty target of messages accepted by this
	// node. The higher this value, the more difficult the Proof of Work must be
	// before this individual will accept the message. This number is the
	// average number of nonce trials a node will have to perform to meet the
	// Proof of Work requirement. 1000 is the network minimum so any lower
	// values will be automatically raised to 1000.
	NonceTrialsPerByte types.Varint
	// Used to calculate the difficulty target of messages accepted by this
	// node. The higher this value, the more difficult the Proof of Work must be
	// before this individual will accept the message. This number is added to
	// the data length to make sending small messages more difficult. 1000 is
	// the network minimum so any lower values will be automatically raised to
	// 1000.
	ExtraBytes types.Varint
	// Length of the signature
	// SigLength types.Varint

	// The ECDSA signature which, as of protocol v3, covers the object header
	// starting with the time, appended with the data described in this table
	// down to the extra_bytes.
	Signature []byte
}

func (obj *PubkeyV3) Serialize() []byte {
	var b bytes.Buffer

	binary.Write(&b, binary.BigEndian, obj.Behaviour)
	b.Write(obj.PubSigningKey[:])
	b.Write(obj.PubEncryptionKey[:])
	b.Write(obj.NonceTrialsPerByte.Serialize())
	b.Write(obj.ExtraBytes.Serialize())
	b.Write(types.Varint(len(obj.Signature)).Serialize())
	b.Write(obj.Signature)

	return b.Bytes()
}

func (obj *PubkeyV3) DeserializeReader(b io.Reader) error {
	err := binary.Read(b, binary.BigEndian, &obj.Behaviour)
	if err != nil {
		return types.DeserializeFailedError("behaviour")
	}
	err = binary.Read(b, binary.BigEndian, obj.PubSigningKey[:])
	if err != nil {
		return types.DeserializeFailedError("pubSigningKey")
	}
	err = binary.Read(b, binary.BigEndian, obj.PubEncryptionKey[:])
	if err != nil {
		return types.DeserializeFailedError("pubEncryptionKey")
	}
	err = obj.NonceTrialsPerByte.DeserializeReader(b)
	if err != nil {
		return types.DeserializeFailedError("nonceTrialsPerByte: " + err.Error())
	}
	err = obj.ExtraBytes.DeserializeReader(b)
	if err != nil {
		return types.DeserializeFailedError("extraBytes: " + err.Error())
	}
	var sigLength types.Varint
	err = sigLength.DeserializeReader(b)
	if err != nil {
		return types.DeserializeFailedError("sigLength: " + err.Error())
	}
	obj.Signature = make([]byte, int(sigLength))
	err = binary.Read(b, binary.BigEndian, obj.Signature)
	if err != nil {
		return types.DeserializeFailedError("signature")
	}

	return nil
}

// When version 4 pubkeys are created, most of the data in the pubkey is
// encrypted. This is done in such a way that only someone who has the
// Bitmessage address which corresponds to a pubkey can decrypt and use that
// pubkey. This prevents people from gathering pubkeys sent around the network
// and using the data from them to create messages to be used in spam or in
// flooding attacks.
type PubkeyEncryptedV4 struct {
	// The tag, made up of bytes 32-64 of the double hash of the address data.
	Tag [32]byte
	// Encrypted pubkey data.
	EncryptedData []byte
}

func (obj *PubkeyEncryptedV4) Serialize() []byte {
	var b bytes.Buffer

	b.Write(obj.Tag[:])
	b.Write(obj.EncryptedData)

	return b.Bytes()
}

func (obj *PubkeyEncryptedV4) DeserializeReader(b io.Reader) error {
	err := binary.Read(b, binary.BigEndian, obj.Tag[:])
	if err != nil {
		return types.DeserializeFailedError("tag")
	}
	err = binary.Read(b, binary.BigEndian, obj.EncryptedData)
	if err != nil {
		return types.DeserializeFailedError("encryptedData")
	}

	return nil
}

// When decrypted, a version 4 pubkey is the same as a verion 3 pubkey.
type PubkeyDecryptedV4 struct {
	PubkeyV3
}