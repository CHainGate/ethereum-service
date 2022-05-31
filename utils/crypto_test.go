package utils

import "testing"

type AesTest struct {
	key              []byte
	clearTextMessage string
	encryptedMessage string
}

func TestEncryptDecrypt(t *testing.T) {
	var aesTest = AesTest{
		key:              []byte("secret_123456789"),
		clearTextMessage: "my clear text message",
	}

	encryptedMessage, err := Encrypt(aesTest.key, aesTest.clearTextMessage)
	if err != nil {
		t.Errorf("Message encryption failed. Error: %s", err.Error())
	}
	if encryptedMessage == aesTest.clearTextMessage {
		t.Errorf("Message encryption is still cleartext. Error: %s", err.Error())
	}
	aesTest.encryptedMessage = encryptedMessage

	decryptedMessage, err := Decrypt(aesTest.key, aesTest.encryptedMessage)
	if err != nil {
		t.Fatalf("Message decryption failed. Error: %s", err.Error())
	}
	if decryptedMessage != aesTest.clearTextMessage {
		t.Errorf("Expected message to be %s, but got %s", aesTest.clearTextMessage, decryptedMessage)
	}
}
