package producer

import "testing"

func TestAdjustHash(t *testing.T) {
	type args struct {
		shardhash string
		buckets   int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"TestAdjustHash_1", args{"127.0.0.1", 1}, "00000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 1}, "00000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 2}, "80000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 2}, "00000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 4}, "c0000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 4}, "40000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 8}, "e0000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 8}, "60000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 16}, "f0000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 16}, "60000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 32}, "f0000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 32}, "68000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 64}, "f4000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 64}, "6c000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 128}, "f4000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 128}, "6e000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"127.0.0.1", 256}, "f5000000000000000000000000000000", false},
		{"TestAdjustHash_1", args{"192.168.0.2", 256}, "6f000000000000000000000000000000", false},
	}
	for _, tt := range tests {
		got, err := AdjustHash(tt.args.shardhash, tt.args.buckets)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. AdjustHash() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("%q. AdjustHash() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBitCount(t *testing.T) {
	type args struct {
		buckets int
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"TestBitCount_1", args{1}, 0, false},
		{"TestBitCount_1", args{2}, 1, false},
		{"TestBitCount_1", args{4}, 2, false},
		{"TestBitCount_1", args{8}, 3, false},
		{"TestBitCount_1", args{16}, 4, false},
		{"TestBitCount_1", args{32}, 5, false},
		{"TestBitCount_1", args{64}, 6, false},
		{"TestBitCount_1", args{128}, 7, false},
		{"TestBitCount_1", args{256}, 8, false},
		{"TestBitCount_1", args{7}, -1, true},
		{"TestBitCount_1", args{10}, -1, true},
		{"TestBitCount_1", args{0}, -1, true},
		{"TestBitCount_1", args{-10}, -1, true},
	}
	for _, tt := range tests {
		got, err := BitCount(tt.args.buckets)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. BitCount() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("%q. BitCount() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
