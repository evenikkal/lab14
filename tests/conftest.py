import sys
import os

_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

sys.path.insert(0, os.path.join(_ROOT, "data"))
sys.path.insert(0, os.path.join(_ROOT, "collector_py"))
sys.path.insert(0, os.path.join(_ROOT, "analyzer"))
