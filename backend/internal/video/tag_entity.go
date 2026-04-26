package video

import "regexp"

type Tag struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;type:varchar(100);not null" json:"name"`
}

type VideoTag struct {
	ID      uint `gorm:"primaryKey"`
	VideoID uint `gorm:"index;not null"`
	TagID   uint `gorm:"index;not null"`
}

var tagRegex = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

func ExtractTags(text string) []string {
	matches := tagRegex.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		tag := m[1]
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}
