package client

import (
	"github.com/pkg/errors"
	"github.com/rai-project/auth/provider"
	"github.com/rai-project/database"
	"github.com/rai-project/database/mongodb"
	upper "upper.io/db.v3"
)

func TeamName(uname string) (string, error) {

	prof, err := provider.New()
	if err != nil {
		return "", err
	}

	ok, err := prof.Verify()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.Errorf("cannot authenticate using the credentials in %v", prof.Options().ProfilePath)
	}

	db, err := mongodb.NewDatabase("rai")
	if err != nil {
		return "", err
	}
	defer db.Close()

	col2, err := NewFa2017Ece408TeamCollection(db)
	if err != nil {
		return "", err
	}

	//Find Details based on userid and current class
	var teams Fa2017Ece408Teams
	cond := upper.Cond{
		"userid":  uname,
		"current": true,
	}

	err = col2.Find(cond, 0, 0, &teams)
	if err != nil {
		return "", err
	}

	if len(teams) > 1 {
		return "", errors.New("More than one Team entry for " + uname + ".")
	}

	if len(teams) == 0 {

		return "", errors.New("No Team entry for " + uname + ".")
	}

	return teams[0].Team.Teamname, nil
}

func NewFa2017Ece408TeamCollection(db database.Database) (*Fa2017Ece408TeamCollection, error) {
	tbl, err := mongodb.NewTable(db, Fa2017Ece408Team{}.TableName())
	if err != nil {
		return nil, err
	}
	tbl.Create(nil)

	return &Fa2017Ece408TeamCollection{
		MongoTable: tbl.(*mongodb.MongoTable),
	}, nil
}

type Fa2017Ece408TeamCollection struct {
	*mongodb.MongoTable
}

type Fa2017Ece408Team struct {
	Team `bson:",inline"`
	//  Inferences []Inference
}

type Team struct {
	Userid   string
	Teamname string
	Class    string
	Current  bool `bson:"current"`
}

func (Fa2017Ece408Team) TableName() string {
	return "teams"
}

type Fa2017Ece408Teams []Fa2017Ece408Team
