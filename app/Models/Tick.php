<?php

namespace App\Models;

use Database\Factories\TickFactory;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Tick extends Model
{
    /** @use HasFactory<TickFactory> */
    use HasFactory;

    public $timestamps = false;

    protected $fillable = [
        'habit_id',
        'tick_date',
        'created_at',
    ];

    /**
     * @return BelongsTo<Habit, $this>
     */
    public function habit(): BelongsTo
    {
        return $this->belongsTo(Habit::class);
    }

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'tick_date' => 'date',
            'created_at' => 'datetime',
        ];
    }
}
