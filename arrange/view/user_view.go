package view

// UserView 用户展示信息
type UserView struct {
	Name             string `json:"name"`
	EmpID            string `json:"empId"`
	PersonalPhotoURL string `json:"personalPhotoUrl"`
}

func NewUserView(name, empID string) *UserView {
	return &UserView{Name: name, EmpID: empID}
}

func NewUserViewFromEmpID(empID string) *UserView {
	return &UserView{EmpID: empID}
}
