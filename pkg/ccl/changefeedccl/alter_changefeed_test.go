// Copyright 2022 The Cockroach Authors.
//
// Licensed as a CockroachDB Enterprise file under the Cockroach Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     https://github.com/cockroachdb/cockroach/blob/master/licenses/CCL.txt

package changefeedccl

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cockroachdb/cockroach/pkg/ccl/changefeedccl/cdctest"
	"github.com/cockroachdb/cockroach/pkg/ccl/changefeedccl/changefeedbase"
	"github.com/cockroachdb/cockroach/pkg/jobs"
	"github.com/cockroachdb/cockroach/pkg/jobs/jobspb"
	"github.com/cockroachdb/cockroach/pkg/keys"
	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/server/telemetry"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog/desctestutils"
	"github.com/cockroachdb/cockroach/pkg/sql/execinfra"
	"github.com/cockroachdb/cockroach/pkg/sql/tests"
	"github.com/cockroachdb/cockroach/pkg/testutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/serverutils"
	"github.com/cockroachdb/cockroach/pkg/testutils/skip"
	"github.com/cockroachdb/cockroach/pkg/testutils/sqlutils"
	"github.com/cockroachdb/cockroach/pkg/util/ctxgroup"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/cockroach/pkg/util/randutil"
	"github.com/cockroachdb/cockroach/pkg/util/syncutil"
	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
)

func TestAlterChangefeedAddTarget(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(1)`)
		assertPayloads(t, testFeed, []string{
			`foo: [1]->{"after": {"a": 1}}`,
		})

		sqlDB.Exec(t, `INSERT INTO bar VALUES(2)`)
		assertPayloads(t, testFeed, []string{
			`bar: [2]->{"after": {"a": 2}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedAddTargetFamily(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING, FAMILY onlya (a), FAMILY onlyb (b))`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo FAMILY onlya`)
		defer closeFeed(t, testFeed)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(1, 'hello')`)
		assertPayloads(t, testFeed, []string{
			`foo.onlya: [1]->{"after": {"a": 1}}`,
		})

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d ADD foo FAMILY onlyb`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(2, 'goodbye')`)
		assertPayloads(t, testFeed, []string{
			`foo.onlyb: [1]->{"after": {"b": "hello"}}`,
			`foo.onlya: [2]->{"after": {"a": 2}}`,
			`foo.onlyb: [2]->{"after": {"b": "goodbye"}}`,
		})
	}

	// TODO: Figure out why this freezes on other sinks (ex: webhook)
	cdcTest(t, testFn, feedTestForceSink("kafka"))
}

func TestAlterChangefeedSwitchFamily(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING, FAMILY onlya (a), FAMILY onlyb (b))`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo FAMILY onlya`)
		defer closeFeed(t, testFeed)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(1, 'hello')`)
		assertPayloads(t, testFeed, []string{
			`foo.onlya: [1]->{"after": {"a": 1}}`,
		})

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d ADD foo FAMILY onlyb DROP foo FAMILY onlya`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(2, 'goodbye')`)
		assertPayloads(t, testFeed, []string{
			`foo.onlyb: [1]->{"after": {"b": "hello"}}`,
			`foo.onlyb: [2]->{"after": {"b": "goodbye"}}`,
		})
	}

	// TODO: Figure out why this freezes on other sinks (ex: cloudstorage)
	cdcTest(t, testFn, feedTestForceSink("kafka"))
}

func TestAlterChangefeedDropTarget(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo, bar`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d DROP bar`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(1)`)
		assertPayloads(t, testFeed, []string{
			`foo: [1]->{"after": {"a": 1}}`,
		})

		sqlDB.Exec(t, `INSERT INTO bar VALUES(2)`)
		assertPayloads(t, testFeed, nil)
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedDropTargetAfterTableDrop(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo, bar WITH on_error='pause'`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		// Drop bar table.  This should cause the job to be paused.
		sqlDB.Exec(t, `DROP TABLE bar`)
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d DROP bar`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(1)`)
		assertPayloads(t, testFeed, []string{
			`foo: [1]->{"after": {"a": 1}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks, feedTestEnterpriseSinks)
}

func TestAlterChangefeedDropTargetFamily(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING, FAMILY onlya (a), FAMILY onlyb (b))`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo FAMILY onlya, foo FAMILY onlyb`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d DROP foo FAMILY onlyb`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES(1, 'hello')`)
		sqlDB.Exec(t, `INSERT INTO foo VALUES(2, 'goodbye')`)
		assertPayloads(t, testFeed, []string{
			`foo.onlya: [1]->{"after": {"a": 1}}`,
			`foo.onlya: [2]->{"after": {"a": 2}}`,
		})

	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedSetDiffOption(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d SET diff`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES (0, 'initial')`)
		assertPayloads(t, testFeed, []string{
			`foo: [0]->{"after": {"a": 0, "b": "initial"}, "before": null}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedUnsetDiffOption(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo WITH diff`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d UNSET diff`, feed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES (0, 'initial')`)
		assertPayloads(t, testFeed, []string{
			`foo: [0]->{"after": {"a": 0, "b": "initial"}}`,
		})
	}

	// TODO: Figure out why this fails on other sinks
	cdcTest(t, testFn, feedTestForceSink("kafka"))
}

func TestAlterChangefeedErrors(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)
		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.ExpectErr(t,
			`could not load job with job id -1`,
			`ALTER CHANGEFEED -1 ADD bar`,
		)

		sqlDB.Exec(t, `ALTER TABLE bar ADD COLUMN b INT`)
		var alterTableJobID jobspb.JobID
		sqlDB.QueryRow(t, `SELECT job_id FROM [SHOW JOBS] WHERE job_type = 'NEW SCHEMA CHANGE'`).Scan(&alterTableJobID)
		sqlDB.ExpectErr(t,
			fmt.Sprintf(`job %d is not changefeed job`, alterTableJobID),
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar`, alterTableJobID),
		)

		sqlDB.ExpectErr(t,
			fmt.Sprintf(`job %d is not paused`, feed.JobID()),
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar`, feed.JobID()),
		)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.ExpectErr(t,
			`pq: target "TABLE baz" does not exist`,
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD baz`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: target "TABLE baz" does not exist`,
			fmt.Sprintf(`ALTER CHANGEFEED %d DROP baz`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: target "TABLE bar" already not watched by changefeed`,
			fmt.Sprintf(`ALTER CHANGEFEED %d DROP bar`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: invalid option "qux"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d SET qux`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: cannot alter option "initial_scan"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d SET initial_scan`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: invalid option "qux"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d UNSET qux`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: cannot alter option "initial_scan"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d UNSET initial_scan`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: cannot alter option "initial_scan_only"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d UNSET initial_scan_only`, feed.JobID()),
		)
		sqlDB.ExpectErr(t,
			`pq: cannot alter option "end_time"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d UNSET end_time`, feed.JobID()),
		)

		sqlDB.ExpectErr(t,
			`cannot unset option "sink"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d UNSET sink`, feed.JobID()),
		)

		sqlDB.ExpectErr(t,
			`pq: invalid option "diff"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar WITH diff`, feed.JobID()),
		)

		sqlDB.ExpectErr(t,
			`pq: cannot specify both "initial_scan" and "no_initial_scan"`,
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar WITH initial_scan, no_initial_scan`, feed.JobID()),
		)

		sqlDB.ExpectErr(t, "pq: changefeed ID must be an INT value: subqueries are not allowed in cdc",
			"ALTER CHANGEFEED (SELECT 1) ADD bar")
		sqlDB.ExpectErr(t, "pq: changefeed ID must be an INT value: could not parse \"two\" as type int",
			"ALTER CHANGEFEED 'two' ADD bar")
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedDropAllTargetsError(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo, bar`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.ExpectErr(t,
			`cannot drop all targets`,
			fmt.Sprintf(`ALTER CHANGEFEED %d DROP foo, bar`, feed.JobID()),
		)
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedTelemetry(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO foo VALUES (1)`)
		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO bar VALUES (1)`)
		sqlDB.Exec(t, `CREATE TABLE baz (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO baz VALUES (1)`)

		// Reset the counts.
		_ = telemetry.GetFeatureCounts(telemetry.Raw, telemetry.ResetCounts)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo, bar WITH diff`)
		defer closeFeed(t, testFeed)
		feed := testFeed.(cdctest.EnterpriseTestFeed)

		require.NoError(t, feed.Pause())

		// The job system clears the lease asyncronously after
		// cancellation. This lease clearing transaction can
		// cause a restart in the alter changefeed
		// transaction, which will lead to different feature
		// counter counts. Thus, we want to wait for the lease
		// clear. However, the lease clear isn't guaranteed to
		// happen, so we only wait a few seconds for it.
		waitForNoLease := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for {
				if ctx.Err() != nil {
					return
				}
				var sessionID []byte
				sqlDB.QueryRow(t, `SELECT claim_session_id FROM system.jobs WHERE id = $1`, feed.JobID()).Scan(&sessionID)
				if sessionID == nil {
					return
				}
				time.Sleep(250 * time.Millisecond)
			}
		}

		waitForNoLease()
		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d DROP bar, foo ADD baz UNSET diff SET resolved, format=json`, feed.JobID()))

		counts := telemetry.GetFeatureCounts(telemetry.Raw, telemetry.ResetCounts)
		require.Equal(t, int32(1), counts[`changefeed.alter`])
		require.Equal(t, int32(1), counts[`changefeed.alter.dropped_targets.2`])
		require.Equal(t, int32(1), counts[`changefeed.alter.added_targets.1`])
		require.Equal(t, int32(1), counts[`changefeed.alter.set_options.2`])
		require.Equal(t, int32(1), counts[`changefeed.alter.unset_options.1`])
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

// The purpose of this test is to ensure that the ALTER CHANGEFEED statement
// does not accidentally redact secret keys in the changefeed details
func TestAlterChangefeedPersistSinkURI(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	const unredactedSinkURI = "null://blah?AWS_ACCESS_KEY_ID=the_secret"

	params, _ := tests.CreateTestServerParams()
	s, rawSQLDB, _ := serverutils.StartServer(t, params)
	sqlDB := sqlutils.MakeSQLRunner(rawSQLDB)
	registry := s.JobRegistry().(*jobs.Registry)
	ctx := context.Background()
	defer s.Stopper().Stop(ctx)

	query := `CREATE TABLE foo (a string)`
	sqlDB.Exec(t, query)

	query = `CREATE TABLE bar (b string)`
	sqlDB.Exec(t, query)

	query = `SET CLUSTER SETTING kv.rangefeed.enabled = true`
	sqlDB.Exec(t, query)

	var changefeedID jobspb.JobID

	doneCh := make(chan struct{})
	defer close(doneCh)
	registry.TestingResumerCreationKnobs = map[jobspb.Type]func(raw jobs.Resumer) jobs.Resumer{
		jobspb.TypeChangefeed: func(raw jobs.Resumer) jobs.Resumer {
			r := fakeResumer{
				done: doneCh,
			}
			return &r
		},
	}

	sqlDB.QueryRow(t, `CREATE CHANGEFEED FOR TABLE foo, bar INTO $1`, unredactedSinkURI).Scan(&changefeedID)

	sqlDB.Exec(t, `PAUSE JOB $1`, changefeedID)
	waitForJobStatus(sqlDB, t, changefeedID, `paused`)

	sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d SET diff`, changefeedID))

	sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, changefeedID))
	waitForJobStatus(sqlDB, t, changefeedID, `running`)

	job, err := registry.LoadJob(ctx, changefeedID)
	require.NoError(t, err)
	details, ok := job.Details().(jobspb.ChangefeedDetails)
	require.True(t, ok)

	require.Equal(t, unredactedSinkURI, details.SinkURI)
}

func TestAlterChangefeedChangeSinkTypeError(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)

		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.ExpectErr(t,
			`pq: New sink type "null" does not match original sink type "kafka". Altering the sink type of a changefeed is disallowed, consider creating a new changefeed instead.`,
			fmt.Sprintf(`ALTER CHANGEFEED %d SET sink = 'null://'`, feed.JobID()),
		)
	}

	cdcTest(t, testFn, feedTestForceSink("kafka"))
}

func TestAlterChangefeedChangeSinkURI(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		registry := s.Server.JobRegistry().(*jobs.Registry)
		ctx := context.Background()

		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo`)
		defer closeFeed(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		newSinkURI := `kafka://new_kafka_uri`

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d SET sink = '%s'`, feed.JobID(), newSinkURI))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
		waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

		job, err := registry.LoadJob(ctx, feed.JobID())
		require.NoError(t, err)
		details, ok := job.Details().(jobspb.ChangefeedDetails)
		require.True(t, ok)

		require.Equal(t, newSinkURI, details.SinkURI)
	}

	cdcTest(t, testFn, feedTestForceSink("kafka"))
}

func TestAlterChangefeedAddTargetErrors(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO foo (a) SELECT * FROM generate_series(1, 1000)`)

		knobs := s.TestingKnobs.
			DistSQL.(*execinfra.TestingKnobs).
			Changefeed.(*TestingKnobs)

		// Ensure Scan Requests are always small enough that we receive multiple
		// resolved events during a backfill
		knobs.FeedKnobs.BeforeScanRequest = func(b *kv.Batch) error {
			b.Header.MaxSpanRequestKeys = 10
			return nil
		}

		// ensure that we do not emit a resolved timestamp
		knobs.FilterSpanWithMutation = func(r *jobspb.ResolvedSpan) bool {
			return true
		}

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo WITH resolved = '100ms'`)

		// Kafka feeds are not buffered, so we have to consume messages.
		g := ctxgroup.WithContext(context.Background())
		g.Go(func() error {
			for {
				_, err := testFeed.Next()
				if err != nil {
					return err
				}
			}
		})
		defer func() {
			closeFeed(t, testFeed)
			_ = g.Wait()
		}()

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.Pause())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO bar VALUES (1), (2), (3)`)
		sqlDB.ExpectErr(t,
			`pq: target "bar" cannot be resolved as of the creation time of the changefeed. Please wait until the high water mark progresses past the creation time of this target in order to add it to the changefeed.`,
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar`, feed.JobID()),
		)

		// allow the changefeed to emit resolved events now
		knobs.FilterSpanWithMutation = func(r *jobspb.ResolvedSpan) bool {
			return false
		}

		require.NoError(t, feed.Resume())

		// Wait for the high water mark to be non-zero.
		testutils.SucceedsSoon(t, func() error {
			registry := s.Server.JobRegistry().(*jobs.Registry)
			job, err := registry.LoadJob(context.Background(), feed.JobID())
			require.NoError(t, err)
			prog := job.Progress()
			if p := prog.GetHighWater(); p != nil && !p.IsEmpty() {
				return nil
			}
			return errors.New("waiting for highwater")
		})

		require.NoError(t, feed.Pause())
		waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

		sqlDB.Exec(t, `CREATE TABLE baz (a INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO baz VALUES (1), (2), (3)`)

		sqlDB.ExpectErr(t,
			`pq: target "baz" cannot be resolved as of the high water mark. Please wait until the high water mark progresses past the creation time of this target in order to add it to the changefeed.`,
			fmt.Sprintf(`ALTER CHANGEFEED %d ADD baz`, feed.JobID()),
		)
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedDatabaseQualifiedNames(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	skip.WithIssue(t, 83946)
	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE DATABASE movr`)
		sqlDB.Exec(t, `CREATE TABLE movr.drivers (id INT PRIMARY KEY, name STRING)`)
		sqlDB.Exec(t, `CREATE TABLE movr.users (id INT PRIMARY KEY, name STRING)`)
		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (1, 'Alice')`,
		)
		sqlDB.Exec(t,
			`INSERT INTO movr.users VALUES (1, 'Bob')`,
		)
		testFeed := feed(t, f, `CREATE CHANGEFEED FOR movr.drivers WITH resolved = '100ms', diff`)
		defer closeFeed(t, testFeed)

		assertPayloads(t, testFeed, []string{
			`drivers: [1]->{"after": {"id": 1, "name": "Alice"}, "before": null}`,
		})

		expectResolvedTimestamp(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.Pause())

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d ADD movr.users WITH initial_scan UNSET diff`, feed.JobID()))

		require.NoError(t, feed.Resume())

		assertPayloads(t, testFeed, []string{
			`users: [1]->{"after": {"id": 1, "name": "Bob"}}`,
		})

		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (3, 'Carol')`,
		)

		assertPayloads(t, testFeed, []string{
			`drivers: [3]->{"after": {"id": 3, "name": "Carol"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedDatabaseScope(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE DATABASE movr`)
		sqlDB.Exec(t, `CREATE DATABASE new_movr`)

		sqlDB.Exec(t, `CREATE TABLE movr.drivers (id INT PRIMARY KEY, name STRING)`)
		sqlDB.Exec(t, `CREATE TABLE new_movr.drivers (id INT PRIMARY KEY, name STRING)`)

		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (1, 'Alice')`,
		)
		sqlDB.Exec(t,
			`INSERT INTO new_movr.drivers VALUES (1, 'Bob')`,
		)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR movr.drivers WITH diff`)
		defer closeFeed(t, testFeed)

		assertPayloads(t, testFeed, []string{
			`drivers: [1]->{"after": {"id": 1, "name": "Alice"}, "before": null}`,
		})

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.Pause())

		sqlDB.Exec(t, `USE new_movr`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d DROP movr.drivers ADD drivers WITH initial_scan UNSET diff`, feed.JobID()))

		require.NoError(t, feed.Resume())

		assertPayloads(t, testFeed, []string{
			`drivers: [1]->{"after": {"id": 1, "name": "Bob"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedDatabaseScopeUnqualifiedName(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE DATABASE movr`)
		sqlDB.Exec(t, `CREATE DATABASE new_movr`)

		sqlDB.Exec(t, `CREATE TABLE movr.drivers (id INT PRIMARY KEY, name STRING)`)
		sqlDB.Exec(t, `CREATE TABLE new_movr.drivers (id INT PRIMARY KEY, name STRING)`)

		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (1, 'Alice')`,
		)

		sqlDB.Exec(t, `USE movr`)
		testFeed := feed(t, f, `CREATE CHANGEFEED FOR drivers WITH diff, resolved = '100ms'`)
		defer closeFeed(t, testFeed)

		assertPayloads(t, testFeed, []string{
			`drivers: [1]->{"after": {"id": 1, "name": "Alice"}, "before": null}`,
		})

		expectResolvedTimestamp(t, testFeed)

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.Pause())

		sqlDB.Exec(t, `USE new_movr`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d UNSET diff`, feed.JobID()))

		require.NoError(t, feed.Resume())

		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (2, 'Bob')`,
		)

		assertPayloads(t, testFeed, []string{
			`drivers: [2]->{"after": {"id": 2, "name": "Bob"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedColumnFamilyDatabaseScope(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE DATABASE movr`)
		sqlDB.Exec(t, `CREATE TABLE movr.drivers (id INT PRIMARY KEY, name STRING, FAMILY onlyid (id), FAMILY onlyname (name))`)

		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (1, 'Alice')`,
		)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR movr.drivers WITH diff, split_column_families`)
		defer closeFeed(t, testFeed)

		assertPayloads(t, testFeed, []string{
			`drivers.onlyid: [1]->{"after": {"id": 1}, "before": null}`,
			`drivers.onlyname: [1]->{"after": {"name": "Alice"}, "before": null}`,
		})

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.Pause())

		sqlDB.Exec(t, `USE movr`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d DROP movr.drivers ADD movr.drivers FAMILY onlyid ADD drivers FAMILY onlyname UNSET diff`, feed.JobID()))

		require.NoError(t, feed.Resume())

		sqlDB.Exec(t,
			`INSERT INTO movr.drivers VALUES (2, 'Bob')`,
		)

		assertPayloads(t, testFeed, []string{
			`drivers.onlyid: [2]->{"after": {"id": 2}}`,
			`drivers.onlyname: [2]->{"after": {"name": "Bob"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedAlterTableName(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE DATABASE movr`)
		sqlDB.Exec(t, `CREATE TABLE movr.users (id INT PRIMARY KEY, name STRING)`)
		sqlDB.Exec(t,
			`INSERT INTO movr.users VALUES (1, 'Alice')`,
		)
		testFeed := feed(t, f, `CREATE CHANGEFEED FOR movr.users WITH diff, resolved = '100ms'`)
		defer closeFeed(t, testFeed)

		assertPayloads(t, testFeed, []string{
			`users: [1]->{"after": {"id": 1, "name": "Alice"}, "before": null}`,
		})

		expectResolvedTimestamp(t, testFeed)

		waitForSchemaChange(t, sqlDB, `ALTER TABLE movr.users RENAME TO movr.riders`)

		var tsLogical string
		sqlDB.QueryRow(t, `SELECT cluster_logical_timestamp()`).Scan(&tsLogical)

		ts := parseTimeToHLC(t, tsLogical)

		// ensure that the high watermark has progressed past the time in which the
		// schema change occurred
		testutils.SucceedsSoon(t, func() error {
			resolvedTS, _ := expectResolvedTimestamp(t, testFeed)
			if resolvedTS.Less(ts) {
				return errors.New("waiting for resolved timestamp to progress past the schema change event")
			}
			return nil
		})

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.Pause())

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d UNSET diff`, feed.JobID()))

		require.NoError(t, feed.Resume())

		sqlDB.Exec(t,
			`INSERT INTO movr.riders VALUES (2, 'Bob')`,
		)
		assertPayloads(t, testFeed, []string{
			`users: [2]->{"after": {"id": 2, "name": "Bob"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedAddTargetsDuringSchemaChangeError(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	rnd, _ := randutil.NewPseudoRand()

	testFn := func(t *testing.T, s TestServerWithSystem, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		disableDeclarativeSchemaChangesForTest(t, sqlDB)

		knobs := s.TestingKnobs.
			DistSQL.(*execinfra.TestingKnobs).
			Changefeed.(*TestingKnobs)

		sqlDB.Exec(t, `CREATE TABLE foo(val INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO foo (val) SELECT * FROM generate_series(0, 999)`)

		sqlDB.Exec(t, `CREATE TABLE bar(val INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO bar (val) SELECT * FROM generate_series(0, 999)`)

		// Ensure Scan Requests are always small enough that we receive multiple
		// resolved events during a backfill
		knobs.FeedKnobs.BeforeScanRequest = func(b *kv.Batch) error {
			b.Header.MaxSpanRequestKeys = 10
			return nil
		}

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo WITH resolved = '1s', no_initial_scan`)
		jobFeed := testFeed.(cdctest.EnterpriseTestFeed)
		jobRegistry := s.Server.JobRegistry().(*jobs.Registry)

		// Kafka feeds are not buffered, so we have to consume messages.
		g := ctxgroup.WithContext(context.Background())
		g.Go(func() error {
			for {
				_, err := testFeed.Next()
				if err != nil {
					return err
				}
			}
		})
		defer func() {
			closeFeed(t, testFeed)
			_ = g.Wait()
		}()

		// Helper to read job progress
		loadProgress := func() jobspb.Progress {
			jobID := jobFeed.JobID()
			job, err := jobRegistry.LoadJob(context.Background(), jobID)
			require.NoError(t, err)
			return job.Progress()
		}

		// Ensure initial backfill completes
		testutils.SucceedsSoon(t, func() error {
			prog := loadProgress()
			if p := prog.GetHighWater(); p != nil && !p.IsEmpty() {
				return nil
			}
			return errors.New("waiting for highwater")
		})

		// Pause job and setup overrides to force a checkpoint
		require.NoError(t, jobFeed.Pause())

		var maxCheckpointSize int64 = 100 << 20
		// Checkpoint progress frequently, and set the checkpoint size limit.
		changefeedbase.FrontierCheckpointFrequency.Override(
			context.Background(), &s.Server.ClusterSettings().SV, 10*time.Millisecond)
		changefeedbase.FrontierCheckpointMaxBytes.Override(
			context.Background(), &s.Server.ClusterSettings().SV, maxCheckpointSize)

		// Note the tableSpan to avoid resolved events that leave no gaps
		fooDesc := desctestutils.TestingGetPublicTableDescriptor(
			s.SystemServer.DB(), s.Codec, "d", "foo")
		tableSpan := fooDesc.PrimaryIndexSpan(keys.SystemSQLCodec)

		// FilterSpanWithMutation should ensure that once the backfill begins, the following resolved events
		// that are for that backfill (are of the timestamp right after the backfill timestamp) resolve some
		// but not all of the time, which results in a checkpoint eventually being created
		haveGaps := false
		var backfillTimestamp hlc.Timestamp
		var initialCheckpoint roachpb.SpanGroup
		var foundCheckpoint int32
		knobs.FilterSpanWithMutation = func(r *jobspb.ResolvedSpan) bool {
			// Stop resolving anything after checkpoint set to avoid eventually resolving the full span
			if initialCheckpoint.Len() > 0 {
				return true
			}

			// A backfill begins when the backfill resolved event arrives, which has a
			// timestamp such that all backfill spans have a timestamp of
			// timestamp.Next()
			if r.BoundaryType == jobspb.ResolvedSpan_BACKFILL {
				backfillTimestamp = r.Timestamp
				return false
			}

			// Check if we've set a checkpoint yet
			progress := loadProgress()
			if p := progress.GetChangefeed(); p != nil && p.Checkpoint != nil && len(p.Checkpoint.Spans) > 0 {
				initialCheckpoint.Add(p.Checkpoint.Spans...)
				atomic.StoreInt32(&foundCheckpoint, 1)
			}

			// Filter non-backfill-related spans
			if !r.Timestamp.Equal(backfillTimestamp.Next()) {
				// Only allow spans prior to a valid backfillTimestamp to avoid moving past the backfill
				return !(backfillTimestamp.IsEmpty() || r.Timestamp.LessEq(backfillTimestamp.Next()))
			}

			// Only allow resolving if we definitely won't have a completely resolved table
			if !r.Span.Equal(tableSpan) && haveGaps {
				return rnd.Intn(10) > 7
			}
			haveGaps = true
			return true
		}

		require.NoError(t, jobFeed.Resume())
		sqlDB.Exec(t, `ALTER TABLE foo ADD COLUMN b STRING DEFAULT 'd'`)

		// Wait for a checkpoint to have been set
		testutils.SucceedsSoon(t, func() error {
			if atomic.LoadInt32(&foundCheckpoint) != 0 {
				return nil
			}
			return errors.New("waiting for checkpoint")
		})

		require.NoError(t, jobFeed.Pause())
		waitForJobStatus(sqlDB, t, jobFeed.JobID(), `paused`)

		errMsg := fmt.Sprintf(
			`pq: cannot perform initial scan on newly added targets while the checkpoint is non-empty, please unpause the changefeed and wait until the high watermark progresses past the current value %s to add these targets.`,
			backfillTimestamp.AsOfSystemTime(),
		)

		sqlDB.ExpectErr(t, errMsg, fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar WITH initial_scan`, jobFeed.JobID()))
	}

	cdcTestWithSystem(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedAddTargetsDuringBackfill(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	rnd, _ := randutil.NewTestRand()
	var rndMu syncutil.Mutex
	const maxCheckpointSize = 1 << 20
	const numRowsPerTable = 1000

	testFn := func(t *testing.T, s TestServerWithSystem, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo(val INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO foo (val) SELECT * FROM generate_series(0, $1)`, numRowsPerTable-1)

		sqlDB.Exec(t, `CREATE TABLE bar(val INT PRIMARY KEY)`)
		sqlDB.Exec(t, `INSERT INTO bar (val) SELECT * FROM generate_series(0, $1)`, numRowsPerTable-1)

		fooDesc := desctestutils.TestingGetPublicTableDescriptor(
			s.SystemServer.DB(), s.Codec, "d", "foo")
		fooTableSpan := fooDesc.PrimaryIndexSpan(s.Codec)

		knobs := s.TestingKnobs.
			DistSQL.(*execinfra.TestingKnobs).
			Changefeed.(*TestingKnobs)

		// Ensure Scan Requests are always small enough that we receive multiple
		// resolvedFoo events during a backfill
		knobs.FeedKnobs.BeforeScanRequest = func(b *kv.Batch) error {
			rndMu.Lock()
			defer rndMu.Unlock()
			b.Header.MaxSpanRequestKeys = 1 + rnd.Int63n(100)
			return nil
		}

		// Emit resolved events for the majority of spans. Be extra paranoid and ensure that
		// we have at least 1 span for which we don't emit resolvedFoo timestamp (to force checkpointing).
		haveGaps := false
		knobs.FilterSpanWithMutation = func(r *jobspb.ResolvedSpan) bool {
			rndMu.Lock()
			defer rndMu.Unlock()

			if r.Span.Equal(fooTableSpan) {
				// Do not emit resolved events for the entire table span.
				// We "simulate" large table by splitting single table span into many parts, so
				// we want to resolve those sub-spans instead of the entire table span.
				// However, we have to emit something -- otherwise the entire changefeed
				// machine would not work.
				r.Span.EndKey = fooTableSpan.Key.Next()
				return false
			}
			if haveGaps {
				return rnd.Intn(10) > 7
			}
			haveGaps = true
			return true
		}

		// Checkpoint progress frequently, and set the checkpoint size limit.
		changefeedbase.FrontierCheckpointFrequency.Override(
			context.Background(), &s.Server.ClusterSettings().SV, 1)
		changefeedbase.FrontierCheckpointMaxBytes.Override(
			context.Background(), &s.Server.ClusterSettings().SV, maxCheckpointSize)

		registry := s.Server.JobRegistry().(*jobs.Registry)
		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo WITH resolved = '100ms'`)

		g := ctxgroup.WithContext(context.Background())
		g.Go(func() error {
			// Kafka feeds are not buffered, so we have to consume messages.
			// We just want to ensure that eventually, we get all the rows from foo and bar.
			expectedValues := make([]string, 2*numRowsPerTable)
			for j := 0; j < numRowsPerTable; j++ {
				expectedValues[j] = fmt.Sprintf(`foo: [%d]->{"after": {"val": %d}}`, j, j)
				expectedValues[j+numRowsPerTable] = fmt.Sprintf(`bar: [%d]->{"after": {"val": %d}}`, j, j)
			}
			return assertPayloadsBaseErr(context.Background(), testFeed, expectedValues, false, false)
		})

		defer func() {
			require.NoError(t, g.Wait())
			closeFeed(t, testFeed)
		}()

		jobFeed := testFeed.(cdctest.EnterpriseTestFeed)
		loadProgress := func() jobspb.Progress {
			jobID := jobFeed.JobID()
			job, err := registry.LoadJob(context.Background(), jobID)
			require.NoError(t, err)
			return job.Progress()
		}

		// Wait for non-nil checkpoint.
		testutils.SucceedsSoon(t, func() error {
			progress := loadProgress()
			if p := progress.GetChangefeed(); p != nil && p.Checkpoint != nil && len(p.Checkpoint.Spans) > 0 {
				return nil
			}
			return errors.New("waiting for checkpoint")
		})

		// Pause the job and read and verify the latest checkpoint information.
		require.NoError(t, jobFeed.Pause())
		progress := loadProgress()
		require.NotNil(t, progress.GetChangefeed())
		h := progress.GetHighWater()
		noHighWater := h == nil || h.IsEmpty()
		require.True(t, noHighWater)

		jobCheckpoint := progress.GetChangefeed().Checkpoint
		require.Less(t, 0, len(jobCheckpoint.Spans))
		var checkpoint roachpb.SpanGroup
		checkpoint.Add(jobCheckpoint.Spans...)

		waitForJobStatus(sqlDB, t, jobFeed.JobID(), `paused`)

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar WITH initial_scan`, jobFeed.JobID()))

		// Collect spans we attempt to resolve after when we resume.
		var resolvedFoo []roachpb.Span
		knobs.FilterSpanWithMutation = func(r *jobspb.ResolvedSpan) bool {
			if !r.Span.Equal(fooTableSpan) {
				resolvedFoo = append(resolvedFoo, r.Span)
			}
			return false
		}

		require.NoError(t, jobFeed.Resume())

		// Wait for the high water mark to be non-zero.
		testutils.SucceedsSoon(t, func() error {
			prog := loadProgress()
			if p := prog.GetHighWater(); p != nil && !p.IsEmpty() {
				return nil
			}
			return errors.New("waiting for highwater")
		})

		// At this point, highwater mark should be set, and previous checkpoint should be gone.
		progress = loadProgress()
		require.NotNil(t, progress.GetChangefeed())
		require.Equal(t, 0, len(progress.GetChangefeed().Checkpoint.Spans))

		require.NoError(t, jobFeed.Pause())

		// Verify that none of the resolvedFoo spans after resume were checkpointed.
		for _, sp := range resolvedFoo {
			require.Falsef(t, checkpoint.Contains(sp.Key), "span should not have been resolved: %s", sp)
		}
	}

	cdcTestWithSystem(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedUpdateFilter(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	// Skip this test for now.  It used to test alter changefeed with
	// now deprecated and removed 'primary_key_filter' option.
	// Since predicates and projections are no longer a "string" option,
	// alter statement implementation (and grammar) needs to be updated, and
	// this test modified and re-enabled.
	skip.WithIssue(t, 82491)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING)`)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo`)
		defer closeFeed(t, testFeed)

		sqlDB.Exec(t, `INSERT INTO foo  SELECT *, 'initial' FROM generate_series(1, 5)`)
		assertPayloads(t, testFeed, []string{
			`foo: [1]->{"after": {"a": 1, "b": "initial"}}`,
			`foo: [2]->{"after": {"a": 2, "b": "initial"}}`,
			`foo: [3]->{"after": {"a": 3, "b": "initial"}}`,
			`foo: [4]->{"after": {"a": 4, "b": "initial"}}`,
			`foo: [5]->{"after": {"a": 5, "b": "initial"}}`,
		})

		feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		require.NoError(t, feed.TickHighWaterMark(s.Server.Clock().Now()))
		require.NoError(t, feed.Pause())

		// Try to set an invalid filter (column b is not part of primary key).
		sqlDB.ExpectErr(t, "cannot be fully constrained",
			fmt.Sprintf(`ALTER CHANGEFEED %d SET schema_change_policy='stop', primary_key_filter='b IS NULL'`, feed.JobID()))

		// Set filter to emit a > 4.  We expect to see update row 5, and onward.
		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d SET schema_change_policy='stop', primary_key_filter='a > 4'`, feed.JobID()))
		require.NoError(t, feed.Resume())

		// Upsert 10 new values -- we expect to see only 5-10
		sqlDB.Exec(t, `UPSERT INTO foo  SELECT *, 'updated' FROM generate_series(1, 10)`)
		assertPayloads(t, testFeed, []string{
			`foo: [5]->{"after": {"a": 5, "b": "updated"}}`,
			`foo: [6]->{"after": {"a": 6, "b": "updated"}}`,
			`foo: [7]->{"after": {"a": 7, "b": "updated"}}`,
			`foo: [8]->{"after": {"a": 8, "b": "updated"}}`,
			`foo: [9]->{"after": {"a": 9, "b": "updated"}}`,
			`foo: [10]->{"after": {"a": 10, "b": "updated"}}`,
		})

		// Pause again, clear out filter and verify we get expected values.
		require.NoError(t, feed.TickHighWaterMark(s.Server.Clock().Now()))
		require.NoError(t, feed.Pause())

		// Set filter to emit a > 4.  We expect to see update row 5, and onward.
		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d UNSET primary_key_filter`, feed.JobID()))
		require.NoError(t, feed.Resume())

		sqlDB.Exec(t, `UPSERT INTO foo  SELECT *, 'new value' FROM generate_series(1, 10)`)
		assertPayloads(t, testFeed, []string{
			`foo: [1]->{"after": {"a": 1, "b": "new value"}}`,
			`foo: [2]->{"after": {"a": 2, "b": "new value"}}`,
			`foo: [3]->{"after": {"a": 3, "b": "new value"}}`,
			`foo: [4]->{"after": {"a": 4, "b": "new value"}}`,
			`foo: [5]->{"after": {"a": 5, "b": "new value"}}`,
			`foo: [6]->{"after": {"a": 6, "b": "new value"}}`,
			`foo: [7]->{"after": {"a": 7, "b": "new value"}}`,
			`foo: [8]->{"after": {"a": 8, "b": "new value"}}`,
			`foo: [9]->{"after": {"a": 9, "b": "new value"}}`,
			`foo: [10]->{"after": {"a": 10, "b": "new value"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}

func TestAlterChangefeedInitialScan(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	skip.WithIssue(t, 83946)

	testFn := func(initialScanOption string) cdcTestFn {
		return func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
			sqlDB := sqlutils.MakeSQLRunner(s.DB)
			sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY)`)
			sqlDB.Exec(t, `INSERT INTO foo VALUES (1), (2), (3)`)
			sqlDB.Exec(t, `CREATE TABLE bar (a INT PRIMARY KEY)`)
			sqlDB.Exec(t, `INSERT INTO bar VALUES (1), (2), (3)`)

			testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo WITH resolved = '1s', no_initial_scan`)
			defer closeFeed(t, testFeed)

			expectResolvedTimestamp(t, testFeed)

			feed, ok := testFeed.(cdctest.EnterpriseTestFeed)
			require.True(t, ok)

			sqlDB.Exec(t, `PAUSE JOB $1`, feed.JobID())
			waitForJobStatus(sqlDB, t, feed.JobID(), `paused`)

			sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d ADD bar WITH %s`, feed.JobID(), initialScanOption))

			sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, feed.JobID()))
			waitForJobStatus(sqlDB, t, feed.JobID(), `running`)

			expectPayloads := (initialScanOption == "initial_scan = 'yes'" || initialScanOption == "initial_scan")
			if expectPayloads {
				assertPayloads(t, testFeed, []string{
					`bar: [1]->{"after": {"a": 1}}`,
					`bar: [2]->{"after": {"a": 2}}`,
					`bar: [3]->{"after": {"a": 3}}`,
				})
			}

			sqlDB.Exec(t, `INSERT INTO bar VALUES (4)`)
			assertPayloads(t, testFeed, []string{
				`bar: [4]->{"after": {"a": 4}}`,
			})
		}
	}

	for _, initialScanOpt := range []string{
		"initial_scan = 'yes'",
		"initial_scan = 'no'",
		"initial_scan",
		"no_initial_scan"} {
		cdcTest(t, testFn(initialScanOpt), feedTestForceSink("kafka"))
	}
}

// This test checks that the time used to get table descriptors in alter
// changefeed is the time from which changefeed will resume (check
// validateNewTargets for more info on how this time is calculated).
func TestAlterChangefeedWithOldCursorFromCreateChangefeed(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	testFn := func(t *testing.T, s TestServer, f cdctest.TestFeedFactory) {
		registry := s.Server.JobRegistry().(*jobs.Registry)

		sqlDB := sqlutils.MakeSQLRunner(s.DB)
		sqlDB.Exec(t, `CREATE TABLE foo (a INT PRIMARY KEY, b STRING)`)

		var tsLogical string
		sqlDB.QueryRow(t, `SELECT cluster_logical_timestamp()`).Scan(&tsLogical)
		cursor := parseTimeToHLC(t, tsLogical)

		testFeed := feed(t, f, `CREATE CHANGEFEED FOR foo WITH cursor=$1`, tsLogical)
		defer closeFeed(t, testFeed)

		sqlDB.Exec(t, `INSERT INTO foo VALUES (1, 'before')`)
		assertPayloads(t, testFeed, []string{
			`foo: [1]->{"after": {"a": 1, "b": "before"}}`,
		})

		castedFeed, ok := testFeed.(cdctest.EnterpriseTestFeed)
		require.True(t, ok)

		loadProgress := func() jobspb.Progress {
			job, err := registry.LoadJob(context.Background(), castedFeed.JobID())
			require.NoError(t, err)
			return job.Progress()
		}

		testutils.SucceedsSoon(t, func() error {
			progress := loadProgress()
			if hw := progress.GetHighWater(); hw != nil && cursor.LessEq(*hw) {
				return nil
			}
			return errors.New("waiting for checkpoint advance")
		})

		sqlDB.Exec(t, `PAUSE JOB $1`, castedFeed.JobID())
		waitForJobStatus(sqlDB, t, castedFeed.JobID(), `paused`)

		sqlDB.Exec(t, `INSERT INTO foo VALUES (2, 'after')`)

		// Simulate that a significant time has passed since the create
		// change feed command was given - if the highwater mark is not
		// used in the following alter changefeed command, then we will
		// get an error when we try to get a table descriptors using
		// cursor time.
		calculateCursor := func(currentTime *hlc.Timestamp) string {
			return "-3h"
		}
		knobs := s.TestingKnobs.DistSQL.(*execinfra.TestingKnobs).Changefeed.(*TestingKnobs)
		knobs.OverrideCursor = calculateCursor

		sqlDB.Exec(t, fmt.Sprintf(`ALTER CHANGEFEED %d SET format='json'`, castedFeed.JobID()))

		sqlDB.Exec(t, fmt.Sprintf(`RESUME JOB %d`, castedFeed.JobID()))
		waitForJobStatus(sqlDB, t, castedFeed.JobID(), `running`)

		assertPayloads(t, testFeed, []string{
			`foo: [2]->{"after": {"a": 2, "b": "after"}}`,
		})
	}

	cdcTest(t, testFn, feedTestEnterpriseSinks)
}
