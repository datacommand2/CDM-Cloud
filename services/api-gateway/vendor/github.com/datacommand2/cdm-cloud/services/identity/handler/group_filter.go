package handler

import "github.com/jinzhu/gorm"

// groupFilter 는 그룹 목록 검색에 필터를 적용하기위한 인터페이스이다
type groupFilter interface {
	Apply(*gorm.DB) (*gorm.DB, error)
}

// groupNameFilter 는 그룹 이름 필터를 위해 문자열을 전달하는 자료구조이다
type groupNameFilter struct {
	Name string
}

// Apply 그룹 목록 검색에 이름 문자열을 적용하기위한 함수이다
func (f *groupNameFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Where("name LIKE ?", "%"+f.Name+"%"), nil
}
