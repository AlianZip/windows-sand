#version 330 core

layout(location = 0) in vec2 vertexPosition;
uniform vec2 sandflakePosition;

void main() {
    gl_Position = vec4(vertexPosition + sandflakePosition, 0.0, 1.0);
}