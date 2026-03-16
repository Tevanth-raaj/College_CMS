import React, { useState, useEffect, useRef } from 'react'

function SearchBarWithDropdown({
  value = '',
  onChange,
  items = [],
  onSelect,
  onFocus,
  placeholder = "Search...",
  width = "w-full",
  label = null,
  renderItem,
  filterFunction,
  getItemKey,
  disabled = false,
}) {
  const [showDropdown, setShowDropdown] = useState(false)
  const containerRef = useRef(null)

  // Default filter function - searches all string properties
  const defaultFilter = (item, searchTerm) => {
    if (!searchTerm) return true
    const term = searchTerm.toLowerCase()
    return Object.values(item).some(val => 
      String(val).toLowerCase().includes(term)
    )
  }

  // Default key getter
  const defaultGetKey = (item) => {
    if (typeof item === 'string') return item
    return item.id || item.key || item.faculty_id || item.student_id || JSON.stringify(item)
  }

  // Default item renderer
  const defaultRenderItem = (item) => {
    if (typeof item === 'string') {
      return <div className="font-medium text-gray-900">{item}</div>
    }
    return (
      <div className="font-medium text-gray-900">
        {item.name || item.title || item.label || item.faculty_name || item.student_name || item.teacher_name || JSON.stringify(item)}
      </div>
    )
  }

  // Use provided functions or defaults
  const filter = filterFunction || defaultFilter
  const getKey = getItemKey || defaultGetKey
  const render = renderItem || defaultRenderItem

  // Filter items
  const filteredItems = items.filter(item => filter(item, value))

  // Handle click outside
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setShowDropdown(false)
      }
    }

    if (showDropdown) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showDropdown])

  // Handle item selection
  const handleSelect = (item) => {
    if (onSelect) {
      onSelect(item)
    }
    setShowDropdown(false)
  }

  return (
    <div className={width} ref={containerRef}>
      {label && (
        <label className="block text-sm font-medium text-gray-700 ms-2 mb-2">
          {label}
        </label>
      )}
      
      <div className="relative">
        <input
          type="text"
          value={value}
          onChange={(e) => {
            onChange(e)
            setShowDropdown(true)
          }}
          onFocus={(e) => {
            setShowDropdown(true)
            if (onFocus) onFocus(e)
          }}
          disabled={disabled}
          placeholder={placeholder}
          className={`w-full px-3 py-3 border-none bg-background rounded-xl outline-none focus:ring-2 focus:ring-primary focus:border-transparent disabled:bg-gray-100 disabled:cursor-not-allowed transition-all duration-200 placeholder:text-gray-400 ${value ? 'pr-10' : ''}`}
        />
        {value && (
          <button
            type="button"
            onClick={() => {
              onChange({ target: { value: '' } })
              setShowDropdown(false)
            }}
            className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-gray-600"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        )}

        {/* Integrated Dropdown */}
        {showDropdown && !disabled && filteredItems.length > 0 && (
          <div className="absolute z-20 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg max-h-60 overflow-hidden">
            <div className="overflow-y-auto max-h-60">
              {filteredItems.map((item) => (
                <div
                  key={getKey(item)}
                  onClick={() => handleSelect(item)}
                  className="px-3 py-2 cursor-pointer hover:bg-blue-50 border-b border-gray-100 last:border-0 transition-colors"
                >
                  {render(item)}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* No results message */}
        {showDropdown && !disabled && value.trim() !== '' && filteredItems.length === 0 && (
          <div className="absolute z-20 w-full mt-1 bg-white border border-gray-300 rounded-lg shadow-lg">
            <div className="px-3 py-4 text-sm text-gray-500 text-center">
              No results found
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default SearchBarWithDropdown