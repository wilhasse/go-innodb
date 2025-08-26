#!/bin/bash

./go-innodb -file testdata/users/users.ibd -page 4 -sql testdata/users/users.sql -parse -records
