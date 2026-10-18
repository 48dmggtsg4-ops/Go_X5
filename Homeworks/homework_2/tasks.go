package homework

// PairInput соответствует объекту nums/target из условия контеста.
type PairInput struct {
	Nums   []int
	Target int
}

// longestPrefix возвращает самый длинный общий префикс строк.
func longestPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}
	first := []rune(words[0])
	n := len(first)
	for _, s := range words[1:] {
		r := []rune(s)
		if len(r) < n {
			n = len(r)
		}
		i := 0
		for i < n && r[i] == first[i] {
			i++
		}
		n = i
		if n == 0 {
			return ""
		}
	}
	return string(first[:n])
}

// pairIndices возвращает индексы двух разных элементов по возрастанию.
func pairIndices(input PairInput) [2]int {
	res := [2]int{}
	for i := 0; i < len(input.Nums); i++ {
		for j := i + 1; j < len(input.Nums); j++ {
			if i != j && input.Nums[i]+input.Nums[j] == input.Target {
				res[0] = i
				res[1] = j
			}
		}
	}
	return res
}

// hasDuplicate сообщает, встречается ли какое-либо значение повторно.
func hasDuplicate(nums []int) bool {
	if len(nums) <= 1 {
		return false
	}
	flag := false
	for i := 0; i < len(nums); i++ {
		for j := i + 1; j < len(nums); j++ {
			if nums[i] == nums[j] {
				flag = true
			}
		}
	}
	return flag
}

// compactSorted создаёт независимый слайс уникальных значений.
func compactSorted(nums []int) []int {
	if len(nums) == 0 {
		return nil
	}
	res := make([]int, 0, len(nums))
	for i, v := range nums {
		if i == 0 || v != nums[i-1] {
			res = append(res, v)
		}
	}
	return res
}

// balancedBrackets проверяет пары и порядок закрытия скобок.
func balancedBrackets(text string) bool {
	res := []byte{}
	for i := 0; i < len(text); i++ {
		s := text[i]
		if s == '(' || s == '[' || s == '{' {
			res = append(res, s)
		} else {
			if len(res) == 0 {
				return false
			}
			last := res[len(res)-1]
			if (s == ')' && last != '(') || (s == ']' && last != '[') || (s == '}' && last != '{') {
				return false
			}
			res = res[:len(res)-1]
		}
	}
	return len(res) == 0
}
