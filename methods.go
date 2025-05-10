// Copyright Sam Xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelsql

// Method specifics operation in the database/sql package.
type Method string

// Event specifics events in the database/sql package.
type Event string

const (
	// MethodConnectorConnect represents SQL connector connect operation.
	MethodConnectorConnect Method = "sql.connector.connect"
	// MethodConnPing represents SQL connection ping operation.
	MethodConnPing Method = "sql.conn.ping"
	// MethodConnExec represents SQL connection exec operation.
	MethodConnExec Method = "sql.conn.exec"
	// MethodConnQuery represents SQL connection query operation.
	MethodConnQuery Method = "sql.conn.query"
	// MethodConnPrepare represents SQL connection prepare operation.
	MethodConnPrepare Method = "sql.conn.prepare"
	// MethodConnBeginTx represents SQL connection begin transaction operation.
	MethodConnBeginTx Method = "sql.conn.begin_tx"
	// MethodConnResetSession represents SQL connection reset session operation.
	MethodConnResetSession Method = "sql.conn.reset_session"
	// MethodTxCommit represents SQL transaction commit operation.
	MethodTxCommit Method = "sql.tx.commit"
	// MethodTxRollback represents SQL transaction rollback operation.
	MethodTxRollback Method = "sql.tx.rollback"
	// MethodStmtExec represents SQL statement exec operation.
	MethodStmtExec Method = "sql.stmt.exec"
	// MethodStmtQuery represents SQL statement query operation.
	MethodStmtQuery Method = "sql.stmt.query"
	// MethodRows represents SQL rows operation.
	MethodRows Method = "sql.rows"
)

const (
	// EventRowsNext represents the event when a SQL row is accessed via the Next method.
	EventRowsNext Event = "sql.rows.next"
)
