package lib

import "os"

func ConvertText(opts ScramOpts, file *os.File) ([]byte, error) {
	var buffer []byte
	if _, err := file.Read(buffer); err != nil {
		return nil, err
	}
	text := string(buffer)
	if err := BuildDictionary(text); err != nil {
		return nil, err
	}
	reqFrequency := opts.Difficulty*float32(MaxFrequency)
	finalFrequency := opts.DifGradient*float32(MaxFrequency)
	incrementAmount := (finalFrequency - reqFrequency)/float32(len(text))

	for i := range text {
		var j int
		var r rune
		node := DictionaryTree
		for j, r = range text[i:] {
			newNode, ok := node.Next[r]
			if !ok {
				break
			}
			node = newNode
		}
		w, ok := LoadedDictionary[opts.List][text[i:j]]
		if ok && w.Frequency > int(reqFrequency) {
			startText := text[:i]
			endText := text[j+1:]
			text = startText+w.Translations[opts.Lang].W+endText
		}
		reqFrequency+=incrementAmount
	}
	return []byte(text), nil
}
