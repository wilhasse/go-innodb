#!/bin/bash

./go-innodb -file testdata/test.ibd -page 4 -sql testdata/test.sql -parse -records

./go-innodb -file testdata/users/users.ibd -page 4 -sql testdata/users/users.sql -parse -records
