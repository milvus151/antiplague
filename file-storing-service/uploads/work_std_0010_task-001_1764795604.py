import numpy as np
import matplotlib.pyplot as plt
from matplotlib.patches import Circle, Wedge


def draw_complex_system():
    # --- 1. ДАННЫЕ (МАТЕМАТИКА) ---
    # Условие 1: |z - (2 + 3i)| < 3
    c1_x, c1_y = 2, 3
    radius = 3

    # Условие 2: |arg(z - (4 - i))| < 2π/3
    c2_x, c2_y = 4, -1
    angle_rad = 2 * np.pi / 3  # Предел угла в радианах
    angle_deg = np.degrees(angle_rad)  # Предел угла в градусах (120)

    # --- 2. НАСТРОЙКА ХОЛСТА ---
    fig, ax = plt.subplots(figsize=(10, 10))
    ax.set_aspect('equal')  # Чтобы круг был кругом!
    ax.grid(True, linestyle='--', alpha=0.3)

    # --- 3. РИСУЕМ ФИГУРЫ (PATCHES) ---
    # Круг
    circle = Circle((c1_x, c1_y), radius, color='skyblue', alpha=0.4, label='|z - 2 - 3i| < 3')
    ax.add_patch(circle)

    # Сектор (Wedge)
    # Рисуем от -120 до +120 градусов. Радиус берем большой (20), чтобы было видно, что он бесконечный
    sector = Wedge((c2_x, c2_y), 20, -angle_deg, angle_deg, color='salmon', alpha=0.4, label='|arg(z - 4 + i)| < 2π/3')
    ax.add_patch(sector)

    # Доп: рисуем границы сектора линиями для красоты
    # Конец луча 1
    x_ray1 = c2_x + 20 * np.cos(-angle_rad)
    y_ray1 = c2_y + 20 * np.sin(-angle_rad)
    ax.plot([c2_x, x_ray1], [c2_y, y_ray1], 'r--', lw=1)
    # Конец луча 2
    x_ray2 = c2_x + 20 * np.cos(angle_rad)
    y_ray2 = c2_y + 20 * np.sin(angle_rad)
    ax.plot([c2_x, x_ray2], [c2_y, y_ray2], 'r--', lw=1)

    # --- 4. ВЫЧИСЛЕНИЕ ПЕРЕСЕЧЕНИЯ (АЛГОРИТМ) ---
    # Генерируем точки окружности
    t = np.linspace(0, 2 * np.pi, 1000)
    x_circ = c1_x + radius * np.cos(t)
    y_circ = c1_y + radius * np.sin(t)

    x_sol = []
    y_sol = []

    # Проверяем каждую точку окружности: попадает ли она в сектор?
    for i in range(len(t)):
        # Вектор от вершины сектора к текущей точке окружности
        dx = x_circ[i] - c2_x
        dy = y_circ[i] - c2_y

        # Вычисляем угол этого вектора (самое важное место!)
        curr_angle = np.arctan2(dy, dx)

        # arctan2 возвращает (-pi, pi). У нас диапазон симметричный < 2pi/3.
        # Просто проверяем модуль.
        if abs(curr_angle) < angle_rad:
            x_sol.append(x_circ[i])
            y_sol.append(y_circ[i])

    # --- 5. РИСУЕМ РЕШЕНИЕ ---
    if len(x_sol) > 0:
        ax.plot(x_sol, y_sol, color='green', linewidth=4, label='Решение (пересечение)')

    # Точки центров
    ax.plot(c1_x, c1_y, 'bo', label='Центр круга (2+3i)')
    ax.plot(c2_x, c2_y, 'ro', label='Вершина угла (4-i)')

    # Оформление осей
    ax.set_xlim(-2, 8)
    ax.set_ylim(-6, 8)
    ax.set_xlabel('Re(z)')
    ax.set_ylabel('Im(z)')
    ax.legend(loc='lower right')
    ax.set_title('Визуализация системы неравенств на комплексной плоскости')

    plt.show()


if __name__ == "__main__":
    draw_complex_system()
