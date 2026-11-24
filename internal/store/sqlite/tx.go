package sqlite

// Transaction support lives on DB.WithTx: methods use db.q() which prefers
// the open *sql.Tx when set. No separate txDB type is required for P0a.
