#[repr(C)]
pub struct Sandflake {
    pub x: f32,     
    pub y: f32,     
    pub speed: f32, 
}

const GRID_SIZE: usize = 1000; 
const CELL_SIZE: f32 = 2.0 / GRID_SIZE as f32; 

static mut GRID: [[bool; GRID_SIZE]; GRID_SIZE] = [[false; GRID_SIZE]; GRID_SIZE];


const MAX_FALL_SPEED: f32 = 1.0;

#[no_mangle]
pub extern "C" fn initialize_sandflakes(count: usize) -> *mut Sandflake {
    let mut sandflakes = Vec::with_capacity(count);
    for _ in 0..count {
        sandflakes.push(Sandflake {
            x: rand::random::<f32>() * 2.0 - 1.0, 
            y: rand::random::<f32>() * 3.0 - 0.5, 
            speed: 0.01, 
        });
    }
    let ptr = sandflakes.as_mut_ptr();
    std::mem::forget(sandflakes);
    ptr
}

#[no_mangle]
pub extern "C" fn update_sandflakes(sandflakes: *mut Sandflake, count: usize, delta_time: f32) {
    let flakes = unsafe { std::slice::from_raw_parts_mut(sandflakes, count) };

    unsafe {
        for flake in flakes.iter_mut() {
            let mut grid_x = ((flake.x + 1.0) / 2.0 * (GRID_SIZE as f32)) as usize;
            let mut grid_y = ((flake.y + 1.0) / 2.0 * (GRID_SIZE as f32)) as usize;
            
            grid_x = grid_x.min(GRID_SIZE - 1);
            grid_y = grid_y.min(GRID_SIZE - 1);
            
            if grid_y == 0 {
                continue;
            }
            
            flake.speed += delta_time * 0.05;
            flake.speed = flake.speed.min(MAX_FALL_SPEED);

            let interpolated_fall = flake.speed * delta_time;


            if !GRID[grid_y - 1][grid_x]{
                GRID[grid_y][grid_x] = false;
                flake.y -= interpolated_fall;
            } else if grid_x > 0 && !GRID[grid_y - 1][grid_x - 1] {

                GRID[grid_y][grid_x] = false;
                flake.x -= CELL_SIZE;
                flake.y -= interpolated_fall;
            } else if grid_x < GRID_SIZE - 1 && !GRID[grid_y - 1][grid_x + 1]{
                
                GRID[grid_y][grid_x] = false;
                flake.x += CELL_SIZE;
                flake.y -= interpolated_fall;
            } else {
                
                GRID[grid_y][grid_x] = true;
            }

            
            grid_x = ((flake.x + 1.0) / 2.0 * (GRID_SIZE as f32)) as usize;
            grid_y = ((flake.y + 1.0) / 2.0 * (GRID_SIZE as f32)) as usize;

            
            grid_x = grid_x.min(GRID_SIZE - 1);
            grid_y = grid_y.min(GRID_SIZE - 1);

            GRID[grid_y][grid_x] = true; 
        }
    }
}

#[no_mangle]
pub extern "C" fn push_sand(grid: *mut Sandflake, count: usize, cursor_x: f32, cursor_y: f32, direction: i32) {
    let flakes = unsafe { std::slice::from_raw_parts_mut(grid, count) };

    // Определяем индекс курсора в сетке
    let grid_x = ((cursor_x + 1.0) / 2.0 * (GRID_SIZE as f32)) as usize;
    let grid_y = ((cursor_y + 1.0) / 2.0 * (GRID_SIZE as f32)) as usize;

    if grid_x >= GRID_SIZE || grid_y >= GRID_SIZE {
        return; // Курсор за пределами экрана
    }

    unsafe {
        // Направление толкания: -1 = влево, 1 = вправо
        let dir = if direction > 0 { 1 } else { -1 };

        // Сдвигаем строку в направлении толкания
        for y in (0..=grid_y).rev() {
            let mut can_move = true;

            // Проверяем возможность перемещения песчинок в строке
            for x in 0..GRID_SIZE {
                if dir == 1 && x == GRID_SIZE - 1 {
                    can_move = false; // Справа нет места
                }
                if dir == -1 && x == 0 {
                    can_move = false; // Слева нет места
                }
                let next_x = if dir == 1 { x + 1 } else { x - 1 };
                if GRID[y][x] && GRID[y][next_x] {
                    can_move = false; // Песчинки упираются друг в друга
                }
            }

            if !can_move {
                break; // Толкание невозможно
            }

            // Перемещаем песчинки в строке
            if dir == 1 {
                for x in (0..GRID_SIZE - 1).rev() {
                    GRID[y][x + 1] = GRID[y][x];
                    GRID[y][x] = false;
                }
            } else {
                for x in 1..GRID_SIZE {
                    GRID[y][x - 1] = GRID[y][x];
                    GRID[y][x] = false;
                }
            }

            // Обновляем положение песчинок
            for flake in flakes.iter_mut() {
                let flake_x = ((flake.x + 1.0) / 2.0 * GRID_SIZE as f32) as usize;
                let flake_y = ((flake.y + 1.0) / 2.0 * GRID_SIZE as f32) as usize;
                if flake_y == y {
                    flake.x += dir as f32 * CELL_SIZE;
                }
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn free_sandflakes(sandflakes: *mut Sandflake) {
    if sandflakes.is_null() {
        return;
    }
    unsafe {
        let _ = Vec::from_raw_parts(sandflakes, 0, 0);
    }
}