
[] Fix finding token
[] Set token in browser side
[] Add validations for request and model structs
[] Refactor the errors
[] For stream token, use fileID from client side
[] For file chunks get only `chunks` number of items and check the next_chunk_id == -1
[] Try to render the file by concatenating them
[] Make update file chunks work
[] Adding indexes (Try to write a query parser and verify indexes with respect to table props)
  [] The parser will check for `WHERE`, `JOIN`, `DISTINCT`, `ORDER BY`, `GROUP`, `HAVING`
[] Add caching layer

