package strategy

import "reflect"

type User struct {
	ID        string
	Interests []string
}

type MatchingStrategy struct {
	users []*User
}

func NewMatchingStrategy(users []*User) *MatchingStrategy {
	return &MatchingStrategy{
		users: users,
	}
}

func (ms *MatchingStrategy) FindMatchingGroups() ([][]*User, [][]string) {
	var matchingUsers [][]*User
	var commonInterests [][]string

	for groupSize := 3; groupSize <= 6; groupSize++ {
		for i := 0; i < len(ms.users); i++ {
			groups, interests := generateGroup(ms.users, i)
			for j := range groups {
				if !contains(matchingUsers, groups[j]) {
					matchingUsers = append(matchingUsers, groups[j])
					commonInterests = append(commonInterests, interests[j])
				}
			}
		}
	}

	return matchingUsers, commonInterests
}

func generateGroup(users []*User, startIndex int) ([][]*User, [][]string) {
	var matchingUsers [][]*User
	var commonInterests [][]string

	if len(users) < 3 {
		return matchingUsers, commonInterests
	}

	for i := startIndex; i < len(users); i++ {
		for j := i + 1; j < len(users); j++ {
			for k := j + 1; k < len(users); k++ {
				usersSubset := []*User{users[i], users[j], users[k]}
				common := intersectAll(usersSubset)
				if len(common) > 0 {
					matchingUsers = append(matchingUsers, usersSubset)
					commonInterests = append(commonInterests, common)
				}
			}
		}
	}
	return matchingUsers, commonInterests
}

func intersectAll(users []*User) []string {
	intersection := users[0].Interests
	for _, user := range users[1:] {
		intersection = intersect(intersection, user.Interests)
	}
	return intersection
}

func intersect(slice1, slice2 []string) []string {
	var intersection []string
	for _, interest := range slice1 {
		for _, interest2 := range slice2 {
			if interest == interest2 {
				intersection = append(intersection, interest)
				break
			}
		}
	}
	return intersection
}

func contains(groups [][]*User, group []*User) bool {
	for _, g := range groups {
		if reflect.DeepEqual(g, group) {
			return true
		}
	}
	return false
}
