
[x] Fix finding token
[x] Set token in browser side
[] Add validations for request and model structs
[] Refactor the errors
[x] For stream token, use fileID from client side
[x] For file chunks get only `chunks` number of items and check the next_chunk_id == -1
[] Try to render the file by concatenating them
[x] Make update file chunks work
[] Adding indexes (Try to write a query parser and verify indexes with respect to table props)
  [] The parser will check for `WHERE`, `JOIN`, `DISTINCT`, `ORDER BY`, `GROUP`, `HAVING`
[x] Add caching layer
[] Build a scratchpad

