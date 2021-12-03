package reedosolomon

import (
	"log"
)

// RSCodec Reed-Solomon coder/decoder
// ( Кодер-декодер Рида-Соломона )
type RSCodec struct {
	// Primitive - Decimal representation of primitive polynomial to generate lookup table
	// ( Десятичное представление примитивного полинома для создания таблицы поиска )
	Primitive int
	// EccSymbols - Number of additional characters
	// ( Количество дополнительных символов )
	EccSymbols int
}

var (
	exponents = make([]int, 512)
	logs      = make([]int, 256)
)

// InitLookupTables fills exponential & log tables
// ( заполняет экспоненциальные и журнальные таблицы )
func (r *RSCodec) InitLookupTables() {
	// EN //
	// Precompute the logarithm and anti-log tables for faster computation, using the provided primitive polynomial.
	// b**(log_b(x), log_b(y)) == x * y, where b is the base or generator of the logarithm =>
	// we can use any b to precompute logarithm and anti-log tables to use for multiplying two numbers x and y.

	// RUS //
	// Предварительно вычисляем логарифм и антилогарифмические таблицы для более быстрого вычисления, используя предоставленный примитивный полином.
	// b ** (log_b (x), log_b (y)) == x * y, где b - основание или генератор логарифма =>
	// мы можем использовать любое значение b для предварительного вычисления логарифмических и антилогарифмических таблиц, используемых для умножения двух чисел x и y.
	x := 1
	for i := 0; i < 255; i++ {
		exponents[i] = x
		logs[x] = i
		x = russianPeasantMult(x, 2, r.Primitive, 256, true)
	}

	for i := 255; i < 512; i++ {
		exponents[i] = exponents[i-255]
	}
}

// Encode - given message into Reed-Solomon
// ( кодируем данное сообщение кодом Рида-Соломона )
func (r *RSCodec) Encode(data []byte) (encoded []int) {
	byteMessage := make([]int, len(data))
	for i, ch := range data {
		byteMessage[i] = int(ch)
	}

	//if it_key == 0 {
	//	fmt.Println("Original message:", byteMessage)
	//}

	g := rsGeneratorPoly(r.EccSymbols)

	placeholder := make([]int, len(g)-1)
	// Pad the message and divide it by the irreducible generator polynomial
	// Дополнение сообщения и разделиние его на неприводимый порождающий многочлен
	_, remainder := gfPolyDivision(append(byteMessage, placeholder...), g)

	encoded = append(byteMessage, remainder...)
	return
}

// Decode - and correct encoded Reed-Solomon message
// ( Декодирование и коррекция ошибок в сообщении )
func (r *RSCodec) Decode(data []int) ([]int, []int) {
	decoded := data

	if len(data) > 255 {
		log.Fatalf("Message is too long, max allowed size is %d\n", 255)
	}

	synd := calcSyndromes(data, r.EccSymbols)
	if checkSyndromes(synd) {
		m := len(decoded) - r.EccSymbols
		return decoded[:m], decoded[m:]
	}

	// compute the error locator polynomial using Berlekamp-Massey
	// вычисление полинома локатора ошибок с помощью Берлекампа-Масси
	errLoc := unknownErrorLocator(synd, r.EccSymbols)
	// reverse errLoc
	// переворачиваем errLoc
	reverse(errLoc)
	errPos := findErrors(errLoc, len(decoded))

	decoded = correctErrors(decoded, synd, errPos)

	synd = calcSyndromes(decoded, r.EccSymbols)
	if !checkSyndromes(synd) {
		log.Fatalf("Could not correct message\n")
	}

	m := len(decoded) - r.EccSymbols
	return decoded[:m], decoded[m:]
}

func rsGeneratorPoly(nsym int) []int {
	// generate an irreducible polynomial (necessary to encode message in Reed-Solomon)
	// генерация неприводимого многочлена (необходимо для кодирования сообщения по Риду-Соломону)
	g := []int{1}
	for i := 0; i < nsym; i++ {
		g = gfPolyMultiplication(g, []int{1, gfPow(2, i)})
	}
	return g
}
