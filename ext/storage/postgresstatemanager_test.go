package storage

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// it's a SQL db driver. This is how you do that.
	_ "github.com/lib/pq"
)

type PostgresStateManagerSuite struct {
	*assert.Assertions
	t       *testing.T
	require *require.Assertions
	manager *PostgresStateManager
	db      *sql.DB
	logs    logging.LogSinkController
}

func SetupTest(t *testing.T) *PostgresStateManagerSuite {
	var err error

	t.Helper()

	suite := &PostgresStateManagerSuite{
		t:          t,
		Assertions: assert.New(t),
		require:    require.New(t),
	}

	db := setupDB(t)

	sink, ctrl := logging.NewLogSinkSpy()
	suite.manager = NewPostgresStateManager(db, sink)

	suite.logs = ctrl

	port := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		port = np
	}
	connstr := fmt.Sprintf("dbname=sous_test host=localhost port=%s sslmode=disable", port)
	if suite.db, err = sql.Open("postgres", connstr); err != nil {
		suite.FailNow("Error establishing test-assertion DB connection.", "Error: %v", err)
	}
	return suite
}

func TestPostgresStateManagerWriteState_success(t *testing.T) {
	suite := SetupTest(t)

	s := exampleState()

	suite.require.NoError(suite.manager.WriteState(s, testUser))
	suite.Equal(int64(2), suite.pluckSQL("select count(*) from deployments"))

	assert.Len(t, suite.logs.CallsTo("LogMessage"), 13)
	message := suite.logs.CallsTo("LogMessage")[0].PassedArgs().Get(1).(logging.LogMessage)
	logging.AssertMessageFields(t, message, append(
		append(logging.StandardVariableFields, logging.IntervalVariableFields...), "sous-sql-query", "sous-sql-rows"),
		map[string]interface{}{
			"@loglov3-otl": "sous-sql",
		})

	suite.require.NoError(suite.manager.WriteState(s, testUser))
	// Want to be sure that the deployments history doesn't vacuously grow.
	suite.Equal(int64(2), suite.pluckSQL("select count(*) from deployments"))

	ns, err := suite.manager.ReadState()
	suite.require.NoError(err)

	oldD, err := s.Deployments()
	suite.require.NoError(err)
	newD, err := ns.Deployments()
	suite.require.NoError(err)

	for diff := range oldD.Diff(newD).Pairs {
		switch diff.Kind() {
		default:
			suite.Fail("Difference detected between written and read states", "They are: %s %+#v", diff.Kind(), diff)
		case sous.ModifiedKind:
			suite.Fail("Difference detected between written and read states", "%+#v %+#v", diff, diff.Diffs())

		case sous.SameKind:
		}
	}
}

func (suite *PostgresStateManagerSuite) pluckSQL(sql string) interface{} {
	var v interface{}

	suite.t.Helper()

	row := suite.db.QueryRow(sql)
	err := row.Scan(&v)
	suite.require.NoError(err)

	return v
}
