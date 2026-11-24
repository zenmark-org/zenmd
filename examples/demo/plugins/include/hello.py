import sys


def greet(name):
    return "hello, " + name


if __name__ == "__main__":
    print(greet(sys.argv[1] if len(sys.argv) > 1 else "world"))
