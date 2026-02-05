package main_test

import "fmt"

// 演示 make 和 new 的区别

type Person struct {
	Name string
	Age  int
}

func main() {
	fmt.Println("========== make 的使用 ==========")

	// 1. make 创建 slice（长度 0，容量 0）
	slice1 := make([]int, 0)
	fmt.Printf("make([]int, 0): 类型=%T, 值=%v, 长度=%d, 容量=%d\n",
		slice1, slice1, len(slice1), cap(slice1))
	// 输出: 类型=[]int, 值=[], 长度=0, 容量=0

	// 2. make 创建 slice（长度 3，容量 5）
	slice2 := make([]int, 3, 5)
	fmt.Printf("make([]int, 3, 5): 类型=%T, 值=%v, 长度=%d, 容量=%d\n",
		slice2, slice2, len(slice2), cap(slice2))
	// 输出: 类型=[]int, 值=[0 0 0], 长度=3, 容量=5

	// 3. make 创建 map
	map1 := make(map[string]int)
	map1["age"] = 18
	fmt.Printf("make(map[string]int): 类型=%T, 值=%v\n", map1, map1)
	// 输出: 类型=map[string]int, 值=map[age:18]

	// 4. make 创建 channel
	ch := make(chan int, 2)
	ch <- 1
	ch <- 2
	fmt.Printf("make(chan int, 2): 类型=%T, 可以发送和接收\n", ch)
	fmt.Printf("从 channel 读取: %d, %d\n", <-ch, <-ch)

	fmt.Println("\n========== new 的使用 ==========")

	// 5. new 创建 int 指针
	p1 := new(int)
	fmt.Printf("new(int): 类型=%T, 指针地址=%p, 指向的值=%d\n", p1, p1, *p1)
	*p1 = 42
	fmt.Printf("修改后: %d\n", *p1)
	// 输出: 类型=*int, 指向的值=0

	// 6. new 创建结构体指针
	p2 := new(Person)
	fmt.Printf("new(Person): 类型=%T, 值=%+v\n", p2, p2)
	p2.Name = "张三"
	p2.Age = 18
	fmt.Printf("修改后: %+v\n", p2)
	// 输出: 类型=*main.Person, 值=&{Name: Age:0}

	// 7. new 创建 slice（不推荐！返回指向 nil slice 的指针）
	s := new([]int)
	fmt.Printf("new([]int): 类型=%T, 值=%v, *s=%v\n", s, s, *s)
	// 输出: 类型=*[]int, 值=&[], *s=[]
	// 注意：这是指向 nil slice 的指针，不能直接用！

	fmt.Println("\n========== 常见用法对比 ==========")

	// 8. 创建 slice 的三种方式
	s1 := make([]int, 0) // ✅ 推荐：用 make
	s2 := []int{}        // ✅ 推荐：字面量
	var s3 []int         // ⚠️ nil slice，可以 append 但不推荐
	fmt.Printf("make: %v (nil? %v)\n", s1, s1 == nil)
	fmt.Printf("字面量: %v (nil? %v)\n", s2, s2 == nil)
	fmt.Printf("var: %v (nil? %v)\n", s3, s3 == nil)

	// 9. 创建 map 的两种方式
	m1 := make(map[string]int) // ✅ 推荐：用 make
	m2 := map[string]int{}     // ✅ 推荐：字面量
	fmt.Printf("make map: %v (nil? %v)\n", m1, m1 == nil)
	fmt.Printf("字面量 map: %v (nil? %v)\n", m2, m2 == nil)

	// 10. 创建结构体指针的三种方式
	person1 := &Person{Name: "李四", Age: 20} // ✅ 最推荐：字面量
	person2 := new(Person)                  // ✅ 可以：用 new
	person2.Name = "王五"
	person2.Age = 22
	var person3 Person     // 先创建值
	person3Ptr := &person3 // 再取地址
	person3.Name = "赵六"
	person3.Age = 24
	fmt.Printf("字面量: %+v\n", person1)
	fmt.Printf("new: %+v\n", person2)
	fmt.Printf("&: %+v\n", person3Ptr)

	fmt.Println("\n========== 错误示例（已注释） ==========")

	// ❌ 错误 1：用 new 创建 map 并使用
	// m := new(map[string]int)
	// (*m)["key"] = 1  // panic: assignment to entry in nil map

	// ❌ 错误 2：用 make 创建结构体
	// person := make(Person)  // 编译错误：invalid argument: Person is not a slice, map, or channel

	// ❌ 错误 3：未初始化的 map
	// var m map[string]int
	// m["key"] = 1  // panic: assignment to entry in nil map

	// ✅ 正确：用 make 初始化
	m := make(map[string]int)
	m["key"] = 1
	fmt.Printf("正确的 map 使用: %v\n", m)
}
