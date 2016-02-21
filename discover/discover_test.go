package discover

import (
	"testing"
)

func Test_NodeStatusConvert_1(t *testing.T) {
	node := Node {
		rptStatus:	IMMATURE,
	}
	if rslt := NodeStatusConvert(&node); rslt == "DOWN" {
		t.Log("correct")
	}else {
		t.Error("failed: result: ", rslt)
	}
}

func Test_NodeStatusConvert_2(t *testing.T) {
	node := Node {
		rptStatus:	CREATE,
	}
	if rslt := NodeStatusConvert(&node); rslt == "UP" {
		t.Log("correct")
	}else {
		t.Error("failed: result: ", rslt)
	}
}

func Benchmark_NodeStatusConvert_1(b *testing.B) {
	b.StopTimer()
	node := Node {
		rptStatus:	CREATE,
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		NodeStatusConvert(&node)
	}
}
