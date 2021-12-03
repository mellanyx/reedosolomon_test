package reedosolomon

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"time"
)

func FileToArByte(filename string) ([]byte, error) {
	// return []byte, nil if status OK
	// return nil, err if status FAIL
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Unwrap(err)
	}

	return b, nil
}

func ArByteToFile(ar []byte, name string, ext string, permissions fs.FileMode) error {
	// return nil if status OK
	// return err if status FAIL
	err := ioutil.WriteFile(fmt.Sprintf("%s%s", name, ext), ar, permissions)
	if err != nil {
		return errors.Unwrap(err)
	}

	return nil
}

func CollectArByteFile(arByte []byte, eccsyb int) [][]byte {
	var collectArByte [][]byte

	steps := len(arByte) / (255 - eccsyb)

	for i := 0; i <= steps; i++ {
		if len(arByte) >= (255 - eccsyb) {
			collectArByte = append(collectArByte, arByte[0:(255-eccsyb)])
			arByte = arByte[(255 - eccsyb):]
		} else {
			collectArByte = append(collectArByte, arByte[0:len(arByte)])
			arByte = arByte[len(arByte):]
		}
	}

	return collectArByte
}

func CollectArByteNotEccFile(arByte []byte) [][]byte {
	var collectArByte [][]byte

	steps := len(arByte) / 255

	for i := 0; i <= steps; i++ {
		if len(arByte) >= 255 {
			collectArByte = append(collectArByte, arByte[0:255])
			arByte = arByte[255:]
		} else {
			collectArByte = append(collectArByte, arByte[0:len(arByte)])
			arByte = arByte[len(arByte):]
		}
	}

	return collectArByte
}

// EncodeFile - File bitmap encoding and corruption
// In the first argument, we specify one of the two primitive polynomials in decimal notation (285 or 301).
// In the second argument, we specify the number of additional characters, it is equal to two more than the number of expected errors.
// In the third argument, we pass a multidimensional array of bits.
//
// ( Кодирование и повреждение битового массива файла.
// В первом аргументе указываем один из двух примитивных многочленов в десятичном представлении (285 или 301).
// Во втором аргументе указываем количество добавочных символов, оно равно в двое больше количества предполагаемых ошибок ).
// В третьем аргументе передаем многомерный массив бит. )
func EncodeFile(filePath string, Primitive int, EccSymbols int) {
	start := time.Now()
	// Init RS //
	rs := RSCodec{
		// Мы используем GF(2^8), потому что каждое кодовое слово занимает 8 бит
		// Можно использовать два приводимых многочлена в десятичном представлении
		// 285 обычно используется для QR-кодов
		// 301 обычно используется для Data Matrix
		Primitive: Primitive,

		// EccSymbols - Кол-во добавочных символов
		// Кол-во ошибок, которое код сможет исправить = EccSymbols / 2
		EccSymbols: EccSymbols,
	}

	rs.InitLookupTables()

	// Чтение файла //
	fileExt := filepath.Ext(filePath)

	arByte, err := FileToArByte(filePath)
	if err != nil {
		log.Fatal(err)
	}

	collectArByte := CollectArByteFile(arByte, EccSymbols)

	// Encode //
	var encodeCollectArByte [][]int

	// Итерирование многомерного массива, в котором каждым элементом
	// является массив бит размерностью (255 - EccSymbols)
	// Делается для того, что бы скармливать функции кодирования массивы максимально допустимой длины (255)
	for i := 0; i < len(collectArByte); i++ {
		encoded := rs.Encode(collectArByte[i])

		//if i == 0 {
		//	fmt.Println("Encoded: ", encoded)
		//}

		// Закодированный и поврежденный массив битов
		encodeCollectArByte = append(encodeCollectArByte, encoded)
	}

	//fmt.Println("Encoded FINAL: ", encodeCollectArByte[0])

	arResultByte := UnPackArray(encodeCollectArByte)

	if ArByteToFile(arResultByte, "Encoded_File", fileExt, 0644) == nil {
		fmt.Println("Encoded File wrong!")
	}

	duration := time.Since(start)
	fmt.Println("Runtime: ", duration)
}

func CorruptFile(filePath string, EccSymbols int) {
	start := time.Now()
	// Corrupt the message //
	// ( повреждение сообщения )

	// Чтение файла //
	fileExt := filepath.Ext(filePath)

	arByte, err := FileToArByte(filePath)
	if err != nil {
		log.Fatal(err)
	}

	collectArByte := CollectArByteNotEccFile(arByte)

	var arIntFile [][]int

	for i := 0; i < len(collectArByte); i++ {
		byteMessage := make([]int, len(collectArByte[i]))
		for j, ch := range collectArByte[i] {
			byteMessage[j] = int(ch)
		}

		arIntFile = append(arIntFile, byteMessage)
	}

	var corruptCollectArByte [][]int

	// errors byte
	// ( ошибочные биты )
	for i := 0; i < len(arIntFile); i++ {
		encoded := arIntFile[i]

		// corrupt the message
		// ( повреждение сообщения - ошибочные биты )
		for i := 0; i < (EccSymbols / 2); i++ {
			rand.Seed(time.Now().UnixNano())

			randErr := rand.Intn(len(encoded)-0) + 0

			encoded[randErr] = randErr
		}

		// Поврежденный массив битов
		corruptCollectArByte = append(corruptCollectArByte, encoded)
	}

	//fmt.Println("Corrupted FINAL: ", corruptCollectArByte[0])

	arResultByte := UnPackArray(corruptCollectArByte)

	if ArByteToFile(arResultByte, "Corrupted_File", fileExt, 0644) == nil {
		fmt.Println("Corrupted File wrong!")
	}

	duration := time.Since(start)
	fmt.Println("Runtime: ", duration)
}

// DecodeAndFixCorruptFile - Decoding and recovery of the file bitmap.
// In the first argument, we specify the polynomial used for encoding.
// In the second argument, we indicate the number of additional characters specified during encoding.
// In the third argument, we pass a multidimensional array of bits.
//
// ( Декодирование и восстановление битового массива файла.
// В первом аргументе указываем многочлен используемый при кодировании.
// Во втором аргументе указываем количество добавочных символов, указанное при кодировании.
// In the third argument, we pass the encoded and damaged multidimensional array. )
func DecodeAndFixCorruptFile(filePath string, Primitive int, EccSymbols int) {
	start := time.Now()

	// Init RS //
	rs := RSCodec{
		// Мы используем GF(2^8), потому что каждое кодовое слово занимает 8 бит
		// Можно использовать два приводимых многочлена в десятичном представлении
		// 285 обычно используется для QR-кодов
		// 301 обычно используется для Data Matrix
		Primitive: Primitive,

		// EccSymbols - Кол-во добавочных символов
		// Кол-во ошибок, которое код сможет исправить = EccSymbols / 2
		EccSymbols: EccSymbols,
	}

	rs.InitLookupTables()

	// Чтение файла //
	fileExt := filepath.Ext(filePath)

	arByte, err := FileToArByte(filePath)
	if err != nil {
		log.Fatal(err)
	}

	collectArByte := CollectArByteNotEccFile(arByte)

	var corruptCollectArByte [][]int

	for i := 0; i < len(collectArByte); i++ {
		byteMessage := make([]int, len(collectArByte[i]))
		for j, ch := range collectArByte[i] {
			byteMessage[j] = int(ch)
		}

		corruptCollectArByte = append(corruptCollectArByte, byteMessage)
	}

	var decodedCollectArByte [][]int

	for i := 0; i < len(corruptCollectArByte); i++ {
		decoded, _ := rs.Decode(corruptCollectArByte[i])

		decodedCollectArByte = append(decodedCollectArByte, decoded)
	}

	//fmt.Println("Decoded FINAL: ", decodedCollectArByte[0])

	arResultByte := UnPackArray(decodedCollectArByte)

	if ArByteToFile(arResultByte, "Decoded_File", fileExt, 0644) == nil {
		fmt.Println("Decoded File wrong!")
	}

	duration := time.Since(start)
	fmt.Println("Runtime: ", duration)
}

// UnPackArray - Unpack decodedCollectArByte into one array to create a file
//
// ( Распаковываем decodedCollectArByte в один массив для создания файла )
func UnPackArray(decodedCollectArByte [][]int) []byte {
	var arResultInt []int

	for i := 0; i < len(decodedCollectArByte); i++ {
		arResultInt = append(arResultInt, decodedCollectArByte[i]...)
	}

	arResultByte := make([]byte, len(arResultInt))
	for i, ch := range arResultInt {
		arResultByte[i] = byte(ch)
	}

	return arResultByte
}
