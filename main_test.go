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
