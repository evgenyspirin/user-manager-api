package user

type Request struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	Lastname  string `json:"lastname"`
	BirthDate string `json:"birth_date"`
	Phone     string `json:"phone"`
}
