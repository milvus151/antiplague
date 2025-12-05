#include <algorithm>
#include <iostream>
#include <vector>

struct Node {
  int value;
  int left_child = -1;
  int right_child = -1;
};

struct BinaryTree {
  std::vector<Node> my_tree;

  void insertion(int num) {
    if (my_tree.empty()) {
      my_tree.push_back({num, -1, -1});
      return;
    }
    int compare_index = 0;
    while (true) {
      if (num < my_tree[compare_index].value) {
        if (my_tree[compare_index].left_child == -1) {
          my_tree[compare_index].left_child = my_tree.size();
          my_tree.push_back({num, -1, -1});
          return;
        }
        compare_index = my_tree[compare_index].left_child;
      } else {
        if (my_tree[compare_index].right_child == -1) {
          Node new_node = {num, -1, -1};
          my_tree[compare_index].right_child = my_tree.size();
          my_tree.push_back({num, -1, -1});
          return;
        }
        compare_index = my_tree[compare_index].right_child;
      }
    }
  }

  int check_balance(int index) {
    if (index == -1) {
      return 0;
    }
    int left_height = check_balance(my_tree[index].left_child);
    int right_height = check_balance(my_tree[index].right_child);
    if (left_height == -1 || right_height == -1) {
      return -1;
    }
    if (std::abs(left_height - right_height) > 1) {
      return -1;
    }
    return std::max(left_height, right_height) + 1;
  }
};

int main() {
  std::ios_base::sync_with_stdio(false);
  std::cin.tie(nullptr);
  std::vector<int> numbers;
  int t = -1;
  while (t != 0) {
    std::cin >> t;
    if (t != 0) {
      numbers.push_back(t);
    }
  }
  if (numbers.empty()) {
    std::cout << "YES";
    return 0;
  }
  BinaryTree BT;
  BT.my_tree.reserve(numbers.size());
  for (int i = 0; i < static_cast<int>(numbers.size()); i++) {
    BT.insertion(numbers[i]);
  }
  int flag = BT.check_balance(0);
  if (flag != -1) {
    std::cout << "YES";
  } else {
    std::cout << "NO";
  }
  return 0;
}
