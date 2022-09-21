package main

import (
	"reflect"
	"testing"
)

func Test_getArticle(t *testing.T) {
	type args struct {
		id int64
	}
	tests := []struct {
		name     string
		args     args
		wantData *Article
	}{
		// TODO: Add test cases.
		{name: "get", args: args{id: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotData := getArticle(tt.args.id); !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("getArticle() = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func Test_editArticle(t *testing.T) {
	type args struct {
		data *Article
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "edit", args: args{&Article{
			Id:      10,
			Content: "Content",
			Author:  "xxx",
			Email:   "xxx@qq.com",
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := editArticle(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("editArticle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_articleToDb(t *testing.T) {
	type args struct {
		art string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "write", args: args{art: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			articleToDb(tt.args.art)
		})
	}
}
