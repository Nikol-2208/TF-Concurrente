from flask import Flask, jsonify, request
import json

class Student:
    def __init__(self, name, course, grade):
        self.name = name
        self.course = course
        self.grade = grade
    def serialize(self):
        return {
            "name": self.name,
            "course": self.course,
            "grade": self.grade
        }

def deserialize(dictionary):
    val = Student("", "", "")
    val.name = dictionary.name
    val.course = dictionary.course
    val.grade = dictionary.grade
    return val

listOfStudents = [Student("Gabriel", "Math", "18")]
app = Flask(__name__)

@app.route('/', methods=["GET", "POST"])
def grades():
    if request.method == "POST":
        val = request.get_data(as_text=True)
        try:
            x = json.loads(val)
            listOfStudents.append(x)
        except:
            listOfStudents.append(val)
        return val
    elif request.method == "GET":
        val = []
        for i in listOfStudents:
            if type(i) != dict:
                val.append(i.serialize())
            else:
                val.append(i)
        return jsonify(eqtls=val)

if __name__ == "__main__":
    app.run(debug=True, host="localhost", port=8000)